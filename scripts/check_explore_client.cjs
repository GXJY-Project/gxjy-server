// 隔离执行 B224 的原始探索消费方法，不启动客户端或复制第三方脚本
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const assert = require('node:assert/strict');
const crypto = require('node:crypto');
const root = path.resolve(__dirname, '..');
const source = fs.readFileSync(path.join(root, 'data/assets/bili/analysis/local-client/base224-local/original-index.js'), 'utf8');
assert.equal(crypto.createHash('sha256').update(source).digest('hex'), '89513c054d788a55617a86b4820578019b0135e5f3aeb86692bb76b1af39d5c6');
const fixture = JSON.parse(fs.readFileSync(path.join(root, 'internal/game/testdata/explore_python.json'), 'utf8'));

function methods(module, names, globals) {
  const start = source.indexOf('\n' + module + ': [ function(');
  assert.ok(start >= 0, module);
  const next = source.slice(start + 1).search(/\n\w+: \[ function\(/);
  const text = next < 0 ? source.slice(start) : source.slice(start, start + 1 + next);
  function Constructor() {}
  const context = vm.createContext({ t: Constructor, e: Constructor, console: { log() {} }, ...globals });
  for (const name of names) {
    const match = new RegExp('\\b([et])\\.prototype\\.' + name + ' = function\\(').exec(text);
    assert.ok(match, module + '.' + name);
    const rest = text.slice(match.index);
    const end = rest.slice(1).search(/\n[et]\.prototype\./);
    assert.ok(end >= 0, 'method boundary: ' + name);
    vm.runInContext(rest.slice(0, end + 1), context, { timeout: 1000 });
  }
  return Constructor.prototype;
}

let checked = 0;
for (const sample of fixture.cases) {
  const events = [];
  const sceneMethods = methods('ExploreScene', ['synchronize', 'syncMiniMap'], {
    i: { SceneObjectType: { kPlayer: 'PLAYER' } },
    a: { default: { emit: (name) => events.push(name) } }, n: {},
  });
  const scene = Object.assign(Object.create(sceneMethods), {
    _arrMiniMapObject: [],
    syncEffectNPic() {},
    syncPlayer(id, detail) {
      assert.equal(id, '1');
      assert.ok(fixture.tables.HeroBase[detail.model_id]);
      assert.ok(detail.pos && 'x' in detail.pos && 'y' in detail.pos);
      this.player = detail;
    },
    synchronizeObject(id, detail) {
      assert.ok(fixture.tables.ExploreObject[detail.thing_id]);
      assert.ok(detail.pos);
      this.objects[id] = detail;
    },
    syncPlayerPos(body) { assert.ok(body.pos); },
  });
  const globals = {
    u: { GS_SUCCESS: 0 },
    s: { default: { requireDataByNameIdAndKey(table, id, key) {
      assert.deepEqual([table, id, key], ['BaseConfig', 'Guide_DeliveryScene', 'Config']);
      return { sceneId: 'SyntheticGuide' };
    } } },
    M: { default: { getInstance: () => ({ getValue: () => false, setValue() {} }) } },
    d: { default: { emit: (name) => events.push(name) } },
    p: { EVT_MSG_EXPLORE_SCENEDATA: 'scene' },
  };
  const prototype = methods('ExploreModel', ['requestEnter', 'requestPosition', 'requestSceneMini', 'requestQuitScene', 'synchronizeScene', 'synchronizeScenePos'], globals);
  const model = Object.assign(Object.create(prototype), {
    _worldScene: scene, _needRequestQuitScene: true,
    initCurScene() {}, resetLockedObjectId() {}, clearExplore() {}, resetMoveTimeForPerGrid() {},
    getOpenInfo: () => [], backToWorldMap: () => false,
    syncFogAllLight(id, value) { assert.equal(value, true); },
    syncSceneMiniData(id, body) { scene.syncMiniMap(body); },
  });
  for (const step of sample.steps) {
    const response = step.messages.find((m) => m.command === step.command);
    if (!response) continue;
    const reply = structuredClone(response.payload);
    let continued = 0;
    model.request = (request, command, callback) => {
      assert.equal(command, reply.cmd);
      callback(reply);
    };
    if (step.command === 1852) {
      assert.equal(step.messages[0].command, 1856);
      assert.ok(step.messages[0].payload.explore_dp_map);
      scene.objects = {};
      const original = (step.request && (step.request.msg || step.request)) || {};
      model.requestEnter({ sceneId: reply.scene_id, pos: original.pos || { x: 10, y: 20 } }, () => continued++);
      assert.ok(scene.player);
      assert.ok(events.includes('scene'));
      assert.equal(Object.keys(scene.objects).length, Object.values(reply.units).filter((unit) => unit.type !== 'PLAYER').length);
    } else if (step.command === 1853) {
      assert.equal(reply.pos, undefined);
      model.requestPosition(structuredClone(step.request), () => continued++);
      assert.deepEqual(model._currenScenePos, step.request.pos);
    } else if (step.command === 1854) {
      model.requestSceneMini(reply.scene_id, () => continued++);
      assert.equal(scene._arrMiniMapObject.length, 0);
    } else if (step.command === 1864) {
      model.requestQuitScene(() => continued++);
    } else continue;
    assert.equal(continued, 1);
    checked++;
  }
}
console.log(`B224 original explore methods: ${checked} response checks passed; resource loading and full client startup are outside this harness`);
