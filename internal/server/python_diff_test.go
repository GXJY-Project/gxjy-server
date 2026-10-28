package server

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/GXJY-Project/gxjy-server/controllers"
	gxjycrypto "github.com/GXJY-Project/gxjy-server/internal/crypto"
	"github.com/GXJY-Project/gxjy-server/internal/game"
	"github.com/GXJY-Project/gxjy-server/internal/protocol"
	"github.com/GXJY-Project/gxjy-server/internal/storage"
	"github.com/gorilla/websocket"
)

type pythonFixture struct {
	baselineTables
	HTTP []struct {
		Path     string `json:"path"`
		Body     string `json:"body"`
		Response any    `json:"response"`
	} `json:"http"`
	Cases []struct {
		Name           string                    `json:"name"`
		Initial        map[string]any            `json:"initial"`
		InitialJSON    string                    `json:"initial_json"`
		TablesOverride map[string]map[string]any `json:"tables_override"`
		KeysOverride   map[string][]string       `json:"table_keys_override"`
		Options        struct {
			ClientProfile string `json:"client_profile"`
			Now           int64  `json:"now"`
			Skip          bool   `json:"skip_guide"`
			Replay        bool   `json:"replay_prologue"`
		} `json:"options"`
		Steps []struct {
			Command  uint16         `json:"command"`
			Request  any            `json:"request"`
			Messages []game.Message `json:"messages"`
			State    any            `json:"state"`
			Disk     any            `json:"disk"`
		} `json:"steps"`
	} `json:"cases"`
}

func readPythonFixture(t *testing.T) pythonFixture {
	t.Helper()
	path := os.Getenv("GXJY_DIFF_FIXTURE")
	if path == "" {
		path = "../game/testdata/existing_paths_python.json"
	}
	return readPythonFixturePath(t, path)
}

func readPythonFixturePath(t *testing.T, path string) pythonFixture {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var f pythonFixture
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if err = d.Decode(&f); err != nil {
		t.Fatal(err)
	}
	return f
}
func sameJSON(t *testing.T, label string, got, want any) {
	t.Helper()
	a, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Fatalf("%s 不一致\ngot=%s\nwant=%s", label, a, b)
	}
}

func TestHTTPMatchesPython(t *testing.T) {
	f := readPythonFixture(t)
	handler := controllers.LoginHandler{GateHost: "127.0.0.1", HTTPPort: 8080, WSPort: 8765, Token: "local-token-0001"}
	for _, c := range f.HTTP {
		t.Run(c.Path+"/"+c.Body, func(t *testing.T) {
			got := handler.Response(c.Path, []byte(c.Body), time.Unix(1800000000, 0))
			sameJSON(t, "HTTP", got, c.Response)
		})
	}
}

func TestWebSocketMatchesPython(t *testing.T) {
	f := readPythonFixture(t)
	testWebSocketMatchesPython(t, f)
}

func TestMonthCardWebSocketMatchesPython(t *testing.T) {
	f := readPythonFixturePath(t, "../game/testdata/month_card_python.json")
	testWebSocketMatchesPython(t, f)
}

func TestTasksWebSocketMatchesPython(t *testing.T) {
	f := readPythonFixturePath(t, "../game/testdata/tasks_python.json")
	testWebSocketMatchesPython(t, f)
}

func testWebSocketMatchesPython(t *testing.T, f pythonFixture) {
	t.Helper()
	private, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	key, err := gxjycrypto.ParseRSAPrivatePEM(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(private)}))
	if err != nil {
		t.Fatal(err)
	}
	sessionKey := bytes.Repeat([]byte{7}, 32)
	for _, mode := range []string{"plain", "official"} {
		for _, c := range f.Cases {
			t.Run(mode+"/"+c.Name, func(t *testing.T) {
				store := storage.NewJSONStore(t.TempDir())
				if c.Initial != nil {
					data, _ := json.Marshal(c.Initial)
					if c.InitialJSON != "" {
						data = []byte(c.InitialJSON)
					}
					if err := os.WriteFile(store.Path(), data, 0600); err != nil {
						t.Fatal(err)
					}
				}
				tables := make(map[string]map[string]any, len(f.Tables))
				for name, table := range f.Tables {
					tables[name] = table
				}
				for name, table := range c.TablesOverride {
					tables[name] = table
				}
				keys := make(map[string][]string, len(f.Keys))
				for name, values := range f.Keys {
					keys[name] = values
				}
				for name, values := range c.KeysOverride {
					keys[name] = values
				}
				state, err := NewGameStateFromSource(baselineTables{tables, keys}, store, game.Options{ClientProfile: c.Options.ClientProfile, Now: func() time.Time { return time.Unix(c.Options.Now, 0) }, Choose: func(n int) int { return 0 }, SkipGuide: c.Options.Skip, ReplayPrologue: c.Options.Replay})
				if err != nil {
					t.Fatal(err)
				}
				app, err := New(Config{HTTPAddress: "127.0.0.1:8080", WSAddress: "127.0.0.1:8765", GateHost: "127.0.0.1", HTTPPort: 8080, WSPort: 8765, State: state, RSAKey: key})
				if err != nil {
					t.Fatal(err)
				}
				srv := httptest.NewServer(app.GameHandler())
				defer srv.Close()
				conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/game", nil)
				if err != nil {
					t.Fatal(err)
				}
				defer conn.Close()
				_ = conn.SetReadDeadline(time.Now().Add(20 * time.Second))
				sequence := uint32(0)
				for index, step := range c.Steps {
					request, err := anyProtocolValue(step.Request)
					if err != nil {
						t.Fatal(err)
					}
					var frame []byte
					if mode == "plain" {
						frame, err = protocol.BuildPlainFrame(step.Command, request, uint32(index+1))
					} else {
						encrypted, e := protocol.BuildServerFrame(step.Command, request, uint32(index+1), sessionKey, 1, bytes.NewReader(bytes.Repeat([]byte{byte(index)}, 12)))
						if e != nil {
							t.Fatal(e)
						}
						_, offset, e := protocol.ReadVarint(encrypted, 0)
						if e != nil {
							t.Fatal(e)
						}
						var block []byte
						if index == 0 {
							block, e = rsa.EncryptPKCS1v15(rand.Reader, &private.PublicKey, sessionKey)
							if e != nil {
								t.Fatal(e)
							}
						}
						frame = append([]byte{1}, protocol.WriteVarint(int64(len(block)))...)
						frame = append(frame, block...)
						frame = append(frame, encrypted[offset+1:]...)
					}
					if err != nil {
						t.Fatal(err)
					}
					if err = conn.WriteMessage(websocket.BinaryMessage, frame); err != nil {
						t.Fatal(err)
					}
					for _, want := range step.Messages {
						_, body, e := conn.ReadMessage()
						if e != nil {
							t.Fatalf("command %d: %v", step.Command, e)
						}
						var decoded protocol.DecodedBody
						if mode == "plain" {
							decoded, _, e = protocol.ParsePlainFrame(body)
						} else {
							_, offset, ve := protocol.ReadVarint(body, 0)
							if ve != nil {
								t.Fatal(ve)
							}
							decoded, e = protocol.DecodeClientBody(body[offset+1:], sessionKey)
						}
						if e != nil {
							t.Fatal(e)
						}
						sequence++
						if decoded.Command != want.Command || decoded.Sequence != sequence {
							t.Fatalf("command/sequence=%d/%d want=%d/%d", decoded.Command, decoded.Sequence, want.Command, sequence)
						}
						raw, e := protocolValueJSON(decoded.Message)
						if e != nil {
							t.Fatal(e)
						}
						got, e := decodeRequestJSON(raw)
						if e != nil {
							t.Fatal(e)
						}
						sameJSON(t, "回包", got, want.Payload)
					}
					sameJSON(t, "内存状态", state.runtime.Snapshot(), step.State)
					var disk map[string]any
					if _, err := store.Load(&disk); err != nil {
						t.Fatal(err)
					}
					sameJSON(t, "落盘存档", disk, step.Disk)
				}
			})
		}
	}
}
