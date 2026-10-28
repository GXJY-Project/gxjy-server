// 隔离执行 B224 原方法，不启动客户端、不访问网络、不复制原脚本
const fs = require('node:fs');
const vm = require('node:vm');
const crypto = require('node:crypto');
const path = require('node:path');
const assert = require('node:assert/strict');

const project = path.resolve(__dirname, '..');
const sourcePath = process.argv[2] || path.join(project, 'data/assets/bili/analysis/local-client/base224-local/original-index.js');
const source = fs.readFileSync(sourcePath, 'utf8');
assert.equal(crypto.createHash('sha256').update(source).digest('hex'),
  '89513c054d788a55617a86b4820578019b0135e5f3aeb86692bb76b1af39d5c6');
const lines = source.split(/\r?\n/);
const fixture = JSON.parse(fs.readFileSync(path.join(project, 'internal/game/testdata/login_init_python.json'), 'utf8'));
const response = (sample, cmd) => structuredClone(sample.steps.find((step) => step.command === cmd).messages[0].payload);
function methods(ranges, globals) {
  const context = vm.createContext({ t: function Model() {}, ...globals });
  for (const [start, stop] of ranges) {
    vm.runInContext(lines.slice(start - 1, stop).join('\n'), context, { timeout: 1000 });
  }
  return context.t.prototype;
}
function Base() {}
Base.prototype.synchronize = function() {};
let checked = 0;
for (const sample of fixture.cases) {
  const events = [];
  const emit = (name) => events.push(name);
  const types = { Friend: 0, Recommmend: 1, Search: 2, Apply: 3, PrivateFriend: 4, BlackList: 5 };
  const friend = methods([[270302, 270371]], {
    e: Base, a: types, c: { default: { emit } }, h: { EVT_MSG_UPDATE_BLACK_LIST: 'blacklist', EVT_MSG_UPDATE_LIST: 'friends' },
  });
  const model = Object.assign(Object.create(friend), {
    _Version: 0, _chatData: {},
    requestList() { throw new Error('空列表不应继续分页'); },
    addFriendItem() { throw new Error('空列表不应创建好友'); },
    lookPrivateChatHistory() { throw new Error('空联系人不应查询历史'); },
  });
  for (const [cmd, kind] of [[10101, types.Friend], [10124, types.BlackList], [10129, types.PrivateFriend]]) {
    model.synchronizeList(response(sample, cmd), {}, { start: 1, stop: 20 }, kind);
    checked++;
  }
  assert.deepEqual(events, ['friends', 'blacklist']);

  const guild = methods([[288852, 288866]], { e: Base });
  const guildModel = Object.assign(Object.create(guild), {
    _guildVersion: 0, setHaveGuild(value) { assert.equal(value, false); },
    _guildListModel: {
      clear() {}, setInviteCount(value) { assert.equal(value, 0); }, setCDTime(value) { assert.equal(value, 0); },
      synchronize(value) { assert.equal(Object.keys(value).length, 0); },
    },
  });
  guildModel.synchronize(response(sample, 10151));
  checked++;

  const instance = { default: { getInstance: () => ({}) } };
  const room = methods([[386084, 386100]], {
    C: instance, u: instance, f: instance, w: instance, c: { getFightCheckV: () => 'synthetic' }, G_Config: {},
  });
  let continued = 0;
  const roomModel = Object.assign(Object.create(room), {
    request(request, command, callback) {
      assert.equal(command, 'MSG_GET_PVP_ROOM');
      assert.equal(request.show_error, false);
      callback(response(sample, 2016));
    },
  });
  roomModel.requestRoomInfo(false, (hasRoom) => { assert.equal(hasRoom, false); continued++; });
  assert.equal(continued, 1);
  checked++;

  const real = methods([[116271, 116303], [116505, 116524], [116532, 116547]], {
    e: Base, cc: { log() {} }, h: { GS_SUCCESS: 0 },
    s: { default: { requireRecordById(table, id) { assert.ok(fixture.tables[table][id]); return fixture.tables[table][id]; } } },
    u: { default: { emit } }, g: { EVT_MSG_ARENA_UPDATE: 'arena' },
  });
  let infoRequests = 0;
  const realModel = Object.assign(Object.create(real), {
    arenaBoxList: [], initRewardBox() {}, synchronizeLeve() {}, synchronizeScore(value) { this.score = value; },
    request(request, command, callback) {
      const body = response(sample, 1701);
      if (command === 'MSG_REAL_ARENA_INFO') {
        assert.equal(request.type, 0);
        infoRequests++;
        body.cmd = command;
      } else assert.equal(command, 'MSG_REAL_ARENA_BASIC');
      callback(body);
    },
  });
  realModel.requestArenaSeasonInfo(false);
  assert.equal(infoRequests, 1);
  assert.equal(realModel.score, 1000);
  assert.ok(events.includes('arena'));
  checked += 2;
}
const boardFixture = JSON.parse(fs.readFileSync(path.join(project, 'internal/game/testdata/board_python.json'), 'utf8'));
for (const sample of boardFixture.cases) {
  const events = [];
  const player = methods([[382312, 382345]], {
    cc: { log() {} }, h: { GS_SUCCESS: 0 },
    u: { default: { emit: (name) => events.push(name) } },
    p: { EVT_MSG_BOARD_CHANGE: 'board', EVT_MSG_BACK_ALL_REQ: 'backgrounds' },
  });
  const model = Object.assign(Object.create(player), {
    wife: sample.startup.wife || 'H002', back_now: sample.startup.back_now, mode: sample.startup.mode,
  });
  for (const step of sample.steps) {
    const reply = step.messages.find((message) => message.command === step.command).payload;
    let called = 0;
    model.request = (request, command, callback) => {
      assert.equal(command, step.command === 1006 ? 'MSG_SAVE_BOARD_CHANGE' : 'MSG_REQ_ROLE_BOARD');
      callback(structuredClone(reply));
    };
    if (step.command === 1006) {
      const request = (step.request && (step.request.msg || step.request)) || {};
      const previousMode = model.mode;
      model.requestSaveBoardChange(request.hero_id, request.back_id, request.mode, () => called++);
      assert.equal(model.wife, reply.board.wife);
      assert.equal(model.back_now, reply.board.back_now);
      assert.equal(model.mode, reply.board.mode == null ? previousMode : reply.board.mode);
      assert.equal(events.at(-1), 'board');
    } else {
      model.requestBoardInfo(() => called++);
      assert.deepEqual(Array.from(model.backAll), Object.keys(reply.board_info.back_all));
      assert.equal(events.at(-1), 'backgrounds');
    }
    assert.equal(called, 1);
    checked++;
  }
}
const ftFixture = JSON.parse(fs.readFileSync(path.join(project, 'internal/game/testdata/bili224_ft_initial.json'), 'utf8'));
assert.equal(ftFixture.source_sha256, crypto.createHash('sha256').update(source).digest('hex'));
const together = methods([[261209, 261222], [261242, 261258], [261326, 261334], [261411, 261419]], {
  e: Base, d: { GS_SUCCESS: 0 },
});
const ft = new together.constructor();
const initial = { maxCoin: ft._maxCoin, gainCoin: ft._gainCoin, maxDifficultyId: ft._maxDifficultyId, assistNum: ft._assistNum };
ft.getBossById = () => { throw new Error('初始空态不应构造 Boss'); };
ft.request = (request, command, callback) => {
  assert.equal(Object.keys(request).length, 0);
  assert.equal(command, 'MSG_GET_FT_INFO');
  callback(structuredClone(ftFixture.payload));
};
let ftCallbacks = 0;
ft.requestGetFTInfo(() => ftCallbacks++);
assert.equal(ftCallbacks, 1);
assert.equal(ft.getMaxCoin(), initial.maxCoin);
assert.equal(ft.getGainCoin(), initial.gainCoin);
assert.equal(ft._maxDifficultyId, initial.maxDifficultyId);
assert.equal(ft._assistNum, initial.assistNum);
assert.equal(Object.keys(ft._resetTime).length, 0);
assert.equal(ft._collaborator.length, 0);
assert.equal(ft.activityBossActive(), false);
ft.request = (request, command, callback) => callback({ result: 1 });
ft.requestGetFTInfo(() => ftCallbacks++);
assert.equal(ftCallbacks, 1);
checked += 2;
const rogueFixture = JSON.parse(fs.readFileSync(path.join(project, 'internal/game/testdata/bili224_rogue_initial.json'), 'utf8'));
assert.equal(rogueFixture.source_sha256, crypto.createHash('sha256').update(source).digest('hex'));
const rogueEvents = [];
const rogue = methods([[403801, 403837], [403858, 403918], [403931, 403989], [404618, 404626]], {
  e: Base,
  i: { RoguelickWishType: { kSkill: 'Skill', kWish: 'Wish', kBuff: 'Buff' } },
  y: { GS_SUCCESS: 0 }, u: { EVT_MSG_ROGUE_DATA: 'rogue' },
  g: { default: { emit: (event) => rogueEvents.push(event) } },
});
const rg = new rogue.constructor();
const rogueInitial = [rg.resetTimes, rg._endTime, rg._score, rg._scoreMax, rg._curGold, rg._curHp,
  rg._hpMax, rg._carryHero, rg._curDiffculty, rg._curChapter, rg._curPoint, rg._curRpLimit, rg._maxRpLimit];
rg.request = (request, command, callback) => {
  assert.equal(Object.keys(request).length, 0);
  assert.equal(command, 'MSG_REQUEST_ROGUE_INFO');
  callback(structuredClone(rogueFixture.payload));
};
let rogueCallbacks = 0;
rg.requestRogue(() => rogueCallbacks++);
assert.equal(rogueCallbacks, 1);
assert.deepEqual([rg.resetTimes, rg._endTime, rg._score, rg._scoreMax, rg._curGold, rg._curHp,
  rg._hpMax, rg._carryHero, rg._curDiffculty, rg._curChapter, rg._curPoint, rg._curRpLimit, rg._maxRpLimit], rogueInitial);
for (const key of ['ids', 'heroIds', 'rewardedScore', 'unlockTalent', '_skillHero', '_wishHero', '_rogueHandbookInfos']) {
  assert.equal(rg[key].length, 0);
}
assert.deepEqual(rogueEvents, ['rogue']);
rg.request = (request, command, callback) => callback({ result: 1 });
rg.requestRogue(() => rogueCallbacks++);
assert.equal(rogueCallbacks, 1);
checked += 2;

const homeFixture = JSON.parse(fs.readFileSync(path.join(project, 'internal/game/testdata/home_queries_python.json'), 'utf8'));
for (const sample of homeFixture.cases) {
  for (const [cmd, start, stop, name, invoke, globals] of [
    [1002, 382417, 382422, 'requestPowerResetTime', (model, done) => model.requestPowerResetTime({}, done),
      { cc: { log() {} }, h: { GS_SUCCESS: 0 } }],
    [1307, 172857, 172861, 'requestHangInfo', (model, done) => model.requestHangInfo(done), { l: { GS_SUCCESS: 0 } }],
    [1955, 49311, 49315, 'requestFaceNotices', (model, done) => model.requestFaceNotices({}, done), { y: { GS_SUCCESS: 0 } }],
    [2401, 376815, 376823, 'requestPeakvsBaseInfo', (model, done) => model.requestPeakvsBaseInfo(done), {}],
    [1752, 416259, 416265, 'buyMonthCard', (model, done) => model.buyMonthCard('Active', done), { c: { GS_SUCCESS: 0 } }],
  ]) {
    const model = Object.create(methods([[start, stop]], globals));
    const reply = response(sample, cmd);
    model.setBaseInfo = (value) => { assert.equal(value.actInfo.isActOpen, false); model.baseInfo = value; };
    model.request = (request, command, callback) => { assert.equal(command, reply.cmd); callback(structuredClone(reply)); };
    let calls = 0;
    invoke(model, () => calls++);
    assert.equal(calls, 1, name);
    checked++;
  }
  const mail = Object.assign(Object.create(methods([[339666, 339679], [339727, 339734], [339735, 339746]], {
    cc: { log() {} },
  })), { _items: {}, _itemsCache: {}, _mailList: [] });
  mail.request = (request, command, callback) => {
    assert.equal(command, 'MSG_REQ_MAILS'); callback(response(sample, 1502));
  };
  let mailCallbacks = 0;
  mail.requestMailList(true, 1, 20, () => mailCallbacks++);
  assert.equal(mailCallbacks, 1);
  assert.equal(mail._mailList.length, 0);
  assert.equal(Object.keys(mail._items).length, 0);
  checked++;
  const list = Object.assign(Object.create(methods([[280632, 280667], [280677, 280681]], {
    l: { GS_SUCCESS: 0 }, s: { default: { emit() {} } }, c: { SHOWTIPS: 'tip' },
    n: { default: { get: (key) => key } },
  })), { clear() {}, synchronize(value) { assert.equal(Object.keys(value).length, 0); } });
  for (const [cmd, invoke, expected] of [
    [10152, (done) => list.refreshList(done), 1],
    [10167, (done) => list.requestSearchList('synthetic', done), 0],
    [10175, (done) => list.resetGuildList(done), 1],
    [10157, (done) => list.requestInviteMsg(done), 1],
  ]) {
    list.request = (request, command, callback) => {
      const reply = response(sample, cmd); assert.equal(command, reply.cmd); callback(reply);
    };
    let calls = 0;
    invoke(() => calls++);
    assert.equal(calls, expected);
    checked++;
  }
}
console.log(JSON.stringify({ target: 'bili-224', checked_methods: checked,
  boundary: '社交/房间/竞技场/看板/主界面查询及获授权的协战和 Rogue 初态；不是整客户端验收' }));
