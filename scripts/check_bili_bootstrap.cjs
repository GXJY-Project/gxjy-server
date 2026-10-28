// 离线执行已授权快照的启动和登录模块，所有文件操作与 HTTP 请求由内存替身接收
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const crypto = require('node:crypto');
const assert = require('node:assert/strict');
const { spawnSync } = require('node:child_process');

const project = path.resolve(__dirname, '..');
const directory = path.resolve(process.argv[2] || path.join(project, 'data/assets/bili/analysis/verified'));
const report = path.join(directory, 'bootstrap-check.json');
assert.equal(spawnSync('git', ['check-ignore', '-q', report], { cwd: project }).status, 0,
  '输出目录必须由 Git 忽略');
function verifiedSource(name, hash) {
  const text = fs.readFileSync(path.join(directory, name), 'utf8');
  assert.equal(crypto.createHash('sha256').update(text).digest('hex'), hash, name + ' 版本不匹配');
  return text;
}
const source = verifiedSource('index.js', 'a8a75655aafdf51a7b0cc397e64b9d3efe789367311cc9ae80b69a4b9901cda0');
const main = verifiedSource('apk-main.js', 'e4c8e9aaef55f44eb2922b5621cc960218730a6810335b9bac5c5a804b2886b8');
const settings = verifiedSource('settings.js', 'ba214d381d99b8ca8901ba6cbe2d259487fdf1b40e9f9baa958151dc6d7e6e00');
const silent = { log() {}, warn() {}, error() {} };

function bootstrap(order) {
  const trace = [], pending = new Map();
  const scope = { console: silent, CC_PHYSICS_BUILTIN: false, CC_PHYSICS_CANNON: false };
  scope.window = scope;
  scope.jsb = { fileUtils: {
    getWritablePath: () => '/synthetic/', isDirectoryExist: () => false,
    isFileExist: () => false,
  } };
  scope.cc = {
    AssetManager: { BuiltinBundleName: { INTERNAL: 'internal', RESOURCES: 'resources', MAIN: 'main' } },
    debug: { DebugMode: { INFO: 1, ERROR: 0 } }, macro: {}, sys: { isBrowser: false, isMobile: false },
    view: { enableRetina() {}, resizeWithBrowserSize() {} },
    director: { runSceneImmediate() { trace.push('scene'); } },
    assetManager: {
      init() { trace.push('init'); },
      loadScript(files, cb) { trace.push('scripts'); pending.set('scripts', cb);
        assert.equal(files[0], 'src/assets/Script/Global.js'); },
      loadBundle(name, cb) { trace.push(name); pending.set(name, cb); },
      bundles: { find() { return {
        loadScene(scene, ignored, progress, cb) {
          assert.equal(scene, 'db://assets/Scene/main.fire'); cb(null, {});
        },
      }; } },
    },
    game: { run(options, start) { trace.push('run'); start(); } },
  };
  const context = vm.createContext(scope);
  scope.require = (name) => {
    trace.push(name);
    if (name === 'src/settings.js') vm.runInContext(settings, context, { timeout: 1000 });
    else assert.ok(['src/cocos2d-jsb.js', 'jsb-adapter/jsb-engine.js'].includes(name));
  };
  vm.runInContext(main, context, { timeout: 1000 });
  assert.deepEqual(trace, ['src/settings.js', 'src/cocos2d-jsb.js', 'jsb-adapter/jsb-engine.js',
    'init', 'scripts', 'internal', 'resources']);
  for (let i = 0; i < order.length; i++) {
    pending.get(order[i])(null);
    assert.equal(pending.has('main'), i === order.length - 1,
      'main 必须等待全局脚本和两个基础 bundle 都完成');
  }
  pending.get('main')(null);
  assert.deepEqual(trace.slice(-3), ['main', 'run', 'scene']);
  return trace;
}
const bootTrace = bootstrap(['resources', 'scripts', 'internal']);
bootstrap(['internal', 'resources', 'scripts']);

// 仅求值指定模块的原 factory，不执行 bundle 入口或其他游戏模块
function moduleExports(name, dependencies, context) {
  const marker = name + ': [ function(';
  const start = source.indexOf(marker);
  assert.ok(start >= 0);
  assert.equal(source.indexOf(marker, start + marker.length), -1);
  const rest = source.slice(start + name.length + 2);
  const next = /\n[\w$]+: \[ function\(/.exec(rest);
  assert.ok(next);
  const array = rest.slice(0, next.index).trim().replace(/,$/, '');
  const entry = vm.runInContext('(' + array + ')', context, { timeout: 1000 });
  const module = { exports: {} };
  entry[0]((key) => {
    if (Object.hasOwn(dependencies, key)) return dependencies[key];
    return { default: function() {} };
  }, module, module.exports);
  return module.exports;
}
const requests = [], events = [];
const cc = { _RF: { push() {}, pop() {} }, game: { emit: (...args) => events.push(args) } };
const scope = { console: silent, cc, G_Config: { showServerLog: false }, assert: (v) => assert.ok(v) };
scope.XMLHttpRequest = class {
  constructor() { this.headers = {}; }
  open(method, url) { this.method = method; this.url = url; }
  setRequestHeader(name, value) { this.headers[name] = value; }
  send(body) { requests.push({ method: this.method, url: this.url, body, headers: this.headers }); }
};
const context = vm.createContext(scope);
const eventManager = { default: { emit() {} } };
const http = moduleExports('Http', { '../manager/EventManager': eventManager }, context);
const login = moduleExports('LoginModel', {
  '../../../framework/network/Http': http,
  '../../../framework/manager/EventManager': eventManager,
  '../../statistic/model/StatisticModel': { default: { getInstance: () => ({ sendClient() {} }) } },
  '../../statistic/model/StatisticConfig': { StatisticType: {} },
}, context);
const model = Object.create(login.default.prototype);
const base = { platform: 'synthetic', agent: 'synthetic', local: 'cn', environment: 'test',
  device_os: 'android', base_version: '1.1.224', version: '1.1.281' };
Object.assign(model, {
  initBaseInfo: () => ({ ...base }), getUrl: (route) => 'http://127.0.0.1/' + route,
  getNetworkType: () => 'WIFI', getOpenId: () => 'synthetic-user', getAccessKey: () => 'synthetic-access',
  getChannelToken: () => 'synthetic-channel', getPushToken: () => 'synthetic-push',
  getChannelId: () => 'synthetic-channel-id', getToken: () => 'synthetic-token',
  getCurServer: () => ({ getSecId: () => '1' }), getVersion: () => '1.1.281',
  getSDKName: () => 'synthetic-sdk', getAdChannel: () => '',
  _resourceUrl: 'player/getResource', _loginUrl: 'player/login', _enterUrl: 'player/enter',
});
model.requestLogin();
model.requestServerList();
model.requestGate();
assert.deepEqual(requests.map((r) => r.method), ['POST', 'POST', 'GET']);
assert.equal(requests[0].url, 'http://127.0.0.1/player/getResource');
assert.equal(JSON.parse(requests[0].body).network_type, 'WIFI');
assert.equal(requests[0].headers['Content-Type'], 'application/json');
assert.equal(requests[1].url, 'http://127.0.0.1/player/login');
const account = JSON.parse(requests[1].body);
assert.deepEqual([account.openid, account.access_key, account.channel_token, account.push_token],
  ['synthetic-user', 'synthetic-access', 'synthetic-channel', 'synthetic-push']);
const enter = new URL(requests[2].url);
assert.equal(enter.pathname, '/player/enter');
assert.equal(enter.searchParams.get('token'), 'synthetic-token');
assert.equal(enter.searchParams.get('sec_id'), '1');
assert.equal(enter.searchParams.get('client_version'), '1.1.281');
assert.equal(requests[2].body, null);

model.initLoginInfo({ resources: { cdn: ['http://127.0.0.1/gameres/'], version: '1.1.281.1.1',
  remoteCfg: { synthetic: true } }, maintain: false });
assert.equal(model.getCdnUrl()[0], 'http://127.0.0.1/gameres/');
assert.equal(model.getBasePackageVersion(), '1.1.281.1.1');
assert.equal(events[0][0], 'Event_Merge_Config');
const server = moduleExports('LoginServer', {}, context);
const gate = new server.LoginServer({ sec_id: '1', master_id: '1', followers: [] });
gate.synchronize({ gate_ip: '127.0.0.1', port: 18765, type: 'ws' });
assert.equal(gate.getUrl(), 'ws://127.0.0.1:18765/game');

const reader = moduleExports('JsonReader', {}, context);
const tables = { Test: { _subJsonCount: 1, Same: { value: 'base' } },
  Test_sub1: { Same: { value: 'sub' }, Added: { value: 'added' } } };
const merged = reader.JsonReader.prototype._createJsonObj.call({ _getJsonObj: (name) => tables[name] },
  'Test', 'config/data/');
assert.equal(merged.Same.value, 'sub');
assert.equal(merged.Added.value, 'added');
assert.equal(Object.hasOwn(merged, '_subJsonCount'), false);

async function checkSubBundleLoading() {
  cc.sys = { isNative: true };
  cc.Component = function() {};
  cc._decorator = { ccclass: (type) => type, property: () => () => {} };
  cc.game.on = () => {};
  let existing = new Set(['/synthetic/subBundles/CVSound/config.json', 'assets/CVSound/config.json',
    'assets/live2d/config.json', 'assets/heroVideo/config.json', 'assets/remote/config.json']);
  scope.jsb = { fileUtils: { getWritablePath: () => '/synthetic/',
    isFileExist: (file) => existing.has(file) } };
  let trace = [], loading = false;
  const game = moduleExports('GameManager', { './framework/utils/CommonUtils': {
    loadSubBundleByPromise: (relative) => {
      assert.equal(loading, false, '原脚本必须等待前一个子包加载完成');
      loading = true;
      return Promise.resolve().then(() => { trace.push(relative); loading = false; });
    },
  } }, context);
  const manager = Object.create(game.default.prototype);
  await manager.loadSubBundles();
  assert.deepEqual(trace, ['/synthetic/subBundles/CVSound', 'live2d', 'heroVideo', 'remote']);
  assert.deepEqual(Array.from(scope.G_Config.subBundles), ['CVSound', 'live2d', 'heroVideo']);
  const completeTrace = trace;
  existing = new Set(['/synthetic/subBundles/CVSound/config.json']);
  trace = [];
  await manager.loadSubBundles();
  assert.deepEqual(trace, ['/synthetic/subBundles/CVSound']);
  return completeTrace;
}

checkSubBundleLoading().then((subBundleTrace) => {
  const evidence = { status: 'verified', bootstrap_trace: bootTrace,
    completion_orders_checked: 2, http_methods: requests.map((r) => r.method),
    remote_config_consumed: true, gate_url_fields_checked: true, protocol_subtable_override_checked: true };
  const encoded = JSON.stringify(evidence, null, 2) + '\n';
  if (fs.existsSync(report)) assert.equal(fs.readFileSync(report, 'utf8'), encoded);
  else fs.writeFileSync(report, encoded, { flag: 'wx' });
  const subBundleEvidence = { status: 'verified', subbundle_load_trace: subBundleTrace,
    missing_subbundle_skipped: true, subbundle_download_order: Array.from(scope.G_Config.subBundles) };
  const subBundleReport = path.join(directory, 'subbundle-check.json');
  const subBundleJSON = JSON.stringify(subBundleEvidence, null, 2) + '\n';
  if (fs.existsSync(subBundleReport)) assert.equal(fs.readFileSync(subBundleReport, 'utf8'), subBundleJSON);
  else fs.writeFileSync(subBundleReport, subBundleJSON, { flag: 'wx' });
  console.log(JSON.stringify({ ...evidence, ...subBundleEvidence }));
}).catch((error) => { console.error(error); process.exitCode = 1; });
