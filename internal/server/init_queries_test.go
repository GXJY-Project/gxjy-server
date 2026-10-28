package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	gxjycrypto "github.com/GXJY-Project/gxjy-server/internal/crypto"
	"github.com/GXJY-Project/gxjy-server/internal/game"
	"github.com/GXJY-Project/gxjy-server/internal/protocol"
	"github.com/gorilla/websocket"
)

func TestBoardWebSocketMatchesPython(t *testing.T) {
	testWebSocketMatchesPython(t, readPythonFixturePath(t, "../game/testdata/board_python.json"))
}

func TestHomeQueriesWebSocketMatchesPython(t *testing.T) {
	testWebSocketMatchesPython(t, readPythonFixturePath(t, "../game/testdata/home_queries_python.json"))
}

func TestExploreWebSocketMatchesPython(t *testing.T) {
	testWebSocketMatchesPython(t, readPythonFixturePath(t, "../game/testdata/explore_python.json"))
}

func TestFightTogetherInitialWebSocket(t *testing.T) {
	testBili224InitialWebSocket(t, game.CommandFightTogetherInfo, "MSG_GET_FT_INFO", "bili224_ft_initial.json")
}

func TestRogueInitialWebSocket(t *testing.T) {
	testBili224InitialWebSocket(t, game.CommandRogueInfo, "MSG_REQUEST_ROGUE_INFO", "bili224_rogue_initial.json")
}

func testBili224InitialWebSocket(t *testing.T, command uint16, name, fixture string) {
	t.Helper()
	f := readPythonFixturePath(t, "../game/testdata/board_python.json")
	f.Cases = f.Cases[:1]
	c := &f.Cases[0]
	initial, ok := c.Steps[len(c.Steps)-1].State.(map[string]any)
	if !ok {
		t.Fatal("合成初始状态不是对象")
	}
	var expected struct{ Payload map[string]any }
	data, err := os.ReadFile("../game/testdata/" + fixture)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &expected); err != nil {
		t.Fatal(err)
	}
	c.Name = name + "-initial"
	c.Initial = initial
	c.Options.ClientProfile = "bili-224"
	c.Steps = c.Steps[:1]
	c.Steps[0].Command = command
	c.Steps[0].Request = map[string]any{}
	c.Steps[0].Messages = []game.Message{{Command: command, Payload: expected.Payload}}
	c.Steps[0].State, c.Steps[0].Disk = initial, initial
	if got := (&gameSession{}).name(command); got != name {
		t.Fatalf("%d 缺少内置协议名：%s", command, got)
	}
	testWebSocketMatchesPython(t, f)
}

func TestLoginInitializationWebSocketQueries(t *testing.T) {
	f := readPythonFixturePath(t, "../game/testdata/login_init_python.json")
	for i := range f.Cases {
		// 1702 无原型分支，B224 已确认消费与 1701 相同的 arena_info
		for _, step := range f.Cases[i].Steps {
			if step.Command != game.CommandRealArenaBasic {
				continue
			}
			step.Command = game.CommandRealArenaInfo
			step.Request = map[string]any{"type": 0}
			payload := map[string]any{}
			for key, value := range step.Messages[0].Payload {
				payload[key] = value
			}
			payload["cmd"] = "MSG_REAL_ARENA_INFO"
			step.Messages = []game.Message{{Command: game.CommandRealArenaInfo, Payload: payload}}
			f.Cases[i].Steps = append(f.Cases[i].Steps, step)
			break
		}
	}
	testWebSocketMatchesPython(t, f)
}

func TestDrawPoolOrderSurvivesMessagePack(t *testing.T) {
	f := readPythonFixturePath(t, "../game/testdata/login_init_python.json")
	state, err := NewGameStateFromSource(f.baselineTables, nil, game.Options{})
	if err != nil {
		t.Fatal(err)
	}
	messages, err := state.handle(game.CommandDrawSystem, protocol.Nil())
	if err != nil {
		t.Fatal(err)
	}
	value, err := anyProtocolValue(messages[0].Payload)
	if err != nil {
		t.Fatal(err)
	}
	frame, err := protocol.BuildPlainFrame(game.CommandDrawSystem, value, 1)
	if err != nil {
		t.Fatal(err)
	}
	decoded, _, err := protocol.ParsePlainFrame(frame)
	if err != nil {
		t.Fatal(err)
	}
	pools, found := valueField(decoded.Message, "card_pool")
	if !found || len(pools.Fields) != 2 || pools.Fields[0].Key.Text != "ZZZ_Normal" {
		t.Fatalf("网络编码未保留 normal 池优先：%v", pools)
	}
}

func TestSessionRejectsInvalidMessagePackBeforeDispatch(t *testing.T) {
	for _, mode := range []string{"plain", "encrypted"} {
		t.Run(mode, func(t *testing.T) {
			key := bytes.Repeat([]byte{7}, 32)
			finished := make(chan error, 1)
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				connection, err := upgradeWebSocket(w, r)
				if err != nil {
					finished <- err
					return
				}
				defer connection.close()
				// state 故意为空，解码失败必须在业务分发之前返回
				session := &gameSession{connection: connection, key: key}
				finished <- session.run()
			})
			srv := httptest.NewServer(handler)
			defer srv.Close()
			connection, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http"), nil)
			if err != nil {
				t.Fatal(err)
			}
			defer connection.Close()
			var frame []byte
			if mode == "plain" {
				frame, err = protocol.BuildPlainFrame(game.CommandFriends, protocol.Nil(), 1)
				if err != nil {
					t.Fatal(err)
				}
				frame[len(frame)-1] = 0xc1 // MessagePack 保留字节，不是合法正文
			} else {
				body := []byte{0, 0, 0, 1, byte(game.CommandFriends >> 8), byte(game.CommandFriends & 255), 0xc1}
				nonce := bytes.Repeat([]byte{3}, 12)
				ciphertext, tag, err := gxjycrypto.CC2PEncrypt(key, nonce, body, nil)
				if err != nil {
					t.Fatal(err)
				}
				frame = append([]byte{1, 0}, ciphertext...)
				frame = append(frame, tag...)
				frame = append(frame, nonce...)
			}
			if err := connection.WriteMessage(websocket.BinaryMessage, frame); err != nil {
				t.Fatal(err)
			}
			select {
			case err := <-finished:
				if err == nil || !strings.Contains(err.Error(), "MessagePack 解码失败") {
					t.Fatalf("非法正文没有在分发前失败：%v", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("非法正文未结束会话")
			}
			if err := connection.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
				t.Fatal(err)
			}
			if _, body, err := connection.ReadMessage(); err == nil {
				t.Fatalf("非法正文仍收到响应：%d bytes", len(body))
			}
		})
	}
}
