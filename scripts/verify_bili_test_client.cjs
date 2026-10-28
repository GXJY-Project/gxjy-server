// 复用 B 服原脚本验证本地地址和认证帧，只在内存执行网络模块，不运行游戏入口
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const crypto = require('node:crypto');
const assert = require('node:assert/strict');

const directory = path.resolve(process.argv[2]);
const source = fs.readFileSync(path.join(directory, 'patched-index.js'), 'utf8');
const expectedPublic = crypto.createPublicKey(fs.readFileSync(path.join(directory, 'test-public.pem')))
  .export({ type: 'spki', format: 'der' });
const boundary = source.lastIndexOf('}, {}, [');
assert.ok(boundary > 0);
let randomIndex = 0;
const math = Object.create(Math);
math.random = () => ((randomIndex++ % 16) + 0.25) / 16;
const sandbox = {
  window: {}, G_Config: {}, cc: { _RF: { push() {}, pop() {} } },
  console: { log() {}, warn() {}, error() {} }, Math: math,
  Uint8Array, ArrayBuffer, DataView,
};
sandbox.crypto = { getRandomValues(a) { for (let i = 0; i < a.length; i++) a[i] = (i + 1) & 255; return a; } };
sandbox.window.crypto = sandbox.crypto;
sandbox.self = sandbox;
sandbox.global = sandbox;
vm.createContext(sandbox);
vm.runInContext(source.slice(0, boundary) + '}, {}, []);', sandbox, { timeout: 10000 });
const load = sandbox.window.__require;
load('devConfig');
assert.equal(JSON.stringify(sandbox.G_Config.url), JSON.stringify(['http://127.0.0.1:8080/']));
// 只取原 LoginModel 的 getUrl 方法，依赖仍是原 G_Config，不使用地址替身
const loginStart = source.indexOf('LoginModel: [ function(');
assert.ok(loginStart > 0);
const nextModule = /\n[\w$]+: \[ function\(/.exec(source.slice(loginStart + 1));
assert.ok(nextModule);
const loginSource = source.slice(loginStart, loginStart + 1 + nextModule.index);
const getUrl = /\.prototype\.getUrl = (function\(e\) \{[\s\S]*?\n\});/.exec(loginSource);
assert.ok(getUrl);
const buildUrl = vm.runInContext('(' + getUrl[1] + ')', sandbox, { timeout: 1000 });
const model = { _urlIndex: -1 };
for (const route of ['player/getResource', 'player/login', 'player/enter']) {
  assert.equal(buildUrl.call(model, route), 'http://127.0.0.1:8080/' + route);
}
const clientCrypto = load(75);
const publicEncrypt = clientCrypto.publicEncrypt;
let sessionHex;
clientCrypto.publicEncrypt = (options, data) => {
  const pem = options.key.replace('-----BEGIN PUBLIC KEY-----', '-----BEGIN PUBLIC KEY-----\n')
    .replace('-----END PUBLIC KEY-----', '\n-----END PUBLIC KEY-----');
  const actual = crypto.createPublicKey(pem).export({ type: 'spki', format: 'der' });
  assert.ok(actual.equals(expectedPublic));
  assert.equal(options.padding, clientCrypto.constants.RSA_PKCS1_PADDING);
  assert.equal(data.length, 32);
  sessionHex = data.toString('hex');
  return publicEncrypt(options, data);
};
const pack = load('data_pack');
const msgpack = load(200);
const message = { msg: { openid: 'synthetic-user', token: 'synthetic-token', is_reconnected: false }, seq: 1 };
const packet = Buffer.from(pack.packMsg(101, 1, message));
let position = 1, rsaLength = 0, shift = 0, byte;
do { byte = packet[position++]; rsaLength |= (byte & 127) << shift; shift += 7; } while (byte & 128);
assert.equal(packet[0], 1);
assert.equal(rsaLength, 128);
const rsaBlock = packet.subarray(position, position + rsaLength);
const payload = packet.subarray(position + rsaLength);
const decipher = crypto.createDecipheriv('chacha20-poly1305', Buffer.from(sessionHex, 'hex'), payload.subarray(-12), { authTagLength: 16 });
decipher.setAuthTag(payload.subarray(-28, -12));
const body = Buffer.concat([decipher.update(payload.subarray(0, -28)), decipher.final()]);
assert.equal(body.readUInt32BE(0), 1);
assert.equal(body.readUInt16BE(4), 101);
assert.equal(JSON.stringify(msgpack.decode(body.subarray(6))), JSON.stringify(message));
fs.writeFileSync(path.join(directory, 'synthetic-auth.json'), JSON.stringify({
  rsa_block_hex: rsaBlock.toString('hex'), synthetic_session_hex: sessionHex,
  routing_verified: true, original_client_cc2p_verified: true,
}, null, 2) + '\n', { flag: 'wx' });
console.log('原 LoginModel 地址构造、测试公钥及原 data_pack 的 CC2P 认证帧验证通过');
