// 仅在隔离的 VM 中运行已解码 bundle 的网络模块，不执行游戏入口或连接服务器
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const crypto = require('node:crypto');
const assert = require('node:assert/strict');
const { spawnSync } = require('node:child_process');

const project = path.resolve(__dirname, '..');
const directory = path.resolve(process.argv[2] || path.join(project, 'data/assets/bili/analysis/verified'));
assert.equal(spawnSync('git', ['check-ignore', '-q', path.join(directory, 'wire-vectors.json')],
  { cwd: project }).status, 0, '输出目录必须由 Git 忽略');
const source = fs.readFileSync(path.join(directory, 'index.js'), 'utf8');
const sourceHash = crypto.createHash('sha256').update(source).digest('hex');
assert.equal(sourceHash, 'a8a75655aafdf51a7b0cc397e64b9d3efe789367311cc9ae80b69a4b9901cda0');
const boundary = source.lastIndexOf('}, {}, [');
assert.ok(boundary > 0);
let randomIndex = 0;
const math = Object.create(Math);
math.random = () => ((randomIndex++ % 16) + 0.25) / 16;
const sandbox = {
  window: {}, cc: { _RF: { push() {}, pop() {} } },
  console: { log() {}, warn() {}, error() {} },
  Math: math, Uint8Array, ArrayBuffer, DataView,
};
// 固定随机数据只用于构造合成测试，不能用于真实会话
sandbox.crypto = { getRandomValues(a) { for (let i = 0; i < a.length; i++) a[i] = (i + 1) & 255; return a; } };
sandbox.window.crypto = sandbox.crypto;
sandbox.self = sandbox;
sandbox.global = sandbox;
vm.createContext(sandbox);
vm.runInContext(source.slice(0, boundary) + '}, {}, []);', sandbox, { timeout: 10000 });
const load = sandbox.window.__require;
const clientCrypto = load(75);
const publicEncrypt = clientCrypto.publicEncrypt;
let publicKey;
clientCrypto.publicEncrypt = (options, data) => {
  // 客户端 PEM 无换行，Node 的 OpenSSL 解析器需要标准分行
  const pem = options.key.replace('-----BEGIN PUBLIC KEY-----', '-----BEGIN PUBLIC KEY-----\n')
    .replace('-----END PUBLIC KEY-----', '\n-----END PUBLIC KEY-----');
  publicKey = crypto.createPublicKey(pem);
  assert.equal(options.padding, clientCrypto.constants.RSA_PKCS1_PADDING);
  assert.equal(data.toString('hex'), '0123456789abcdef'.repeat(4));
  return publicEncrypt(options, data);
};
const pack = load('data_pack');
const testKey = Buffer.from('0123456789abcdef'.repeat(4), 'hex');
const msgpack = load(200);

function varint(value) {
  const bytes = [];
  while (value > 127) { bytes.push(128 | (value & 127)); value >>>= 7; }
  return Buffer.from([...bytes, value]);
}

function open(payload) {
  const cipher = payload.subarray(0, -28);
  const tag = payload.subarray(-28, -12);
  const nonce = payload.subarray(-12);
  const decipher = crypto.createDecipheriv('chacha20-poly1305', testKey, nonce, { authTagLength: 16 });
  decipher.setAuthTag(tag);
  return Buffer.concat([decipher.update(cipher), decipher.final()]);
}

const cases = [];
for (const [name, command, sequence, message, rsaLength] of [
  ['first', 101, 1, { msg: { openid: 'synthetic-user', token: 'synthetic-token', is_reconnected: false }, seq: 1 }, 128],
  ['later', 901, 2, { msg: { clientTimestamp: 1700000000000 }, seq: 2 }, 0],
  ['reconnect', 101, 1, { msg: { is_reconnected: true }, seq: 3 }, 128],
]) {
  if (name === 'reconnect') pack.resetUseKeyData();
  const packet = Buffer.from(pack.packMsg(command, sequence, message));
  let position = 1, length = 0, shift = 0, byte;
  do { byte = packet[position++]; length |= (byte & 127) << shift; shift += 7; } while (byte & 128);
  assert.equal(packet[0], 1);
  assert.equal(length, rsaLength);
  const body = open(packet.subarray(position + length));
  assert.equal(body.readUInt32BE(0), sequence);
  assert.equal(body.readUInt16BE(4), command);
  assert.equal(JSON.stringify(msgpack.decode(body.subarray(6))), JSON.stringify(message));
  cases.push({ name, command, sequence, rsa_length: length, message, packet_hex: packet.toString('hex') });
}

const reply = { result: 0, synthetic: true };
const header = Buffer.alloc(6);
header.writeUInt32BE(7);
header.writeUInt16BE(101, 4);
const body = Buffer.concat([header, Buffer.from(msgpack.encode(reply))]);
const nonce = Buffer.from('000102030405060708090a0b', 'hex');
const cipher = crypto.createCipheriv('chacha20-poly1305', testKey, nonce, { authTagLength: 16 });
const payload = Buffer.concat([cipher.update(body), cipher.final(), cipher.getAuthTag(), nonce]);
const frame = Buffer.concat([varint(1 + payload.length), Buffer.from([1]), payload]);
const buffer = (b) => Uint8Array.from(b).buffer;
const unpacked = pack.unpackMsg(buffer(frame));
assert.equal(unpacked.cmdId, 101);
assert.equal(unpacked.seq, 7);
assert.equal(JSON.stringify(unpacked.msg), JSON.stringify(reply));
// 原脚本按 e.slice(t) 返回剩余数据，未把长度前缀本身计入 t
assert.equal(unpacked.remainData.byteLength, 1);
const altered = Buffer.from(frame);
altered[altered.length - 13] ^= 1;
assert.equal(pack.unpackMsg(buffer(altered)).cmdId, null);

const spki = publicKey.export({ type: 'spki', format: 'der' });
const fingerprint = crypto.createHash('sha256').update(spki).digest('hex');
const localKey = crypto.createPublicKey(fs.readFileSync(path.join(project, 'data/keys/gate.pem')));
const localMatches = localKey.export({ type: 'spki', format: 'der' }).equals(spki);
const fallback = JSON.parse(fs.readFileSync(path.join(project, 'data/keys/server_rsa.json'), 'utf8'));
const jwk = publicKey.export({ format: 'jwk' });
const publicModulus = BigInt('0x' + Buffer.from(jwk.n, 'base64url').toString('hex'));
const publicExponent = BigInt('0x' + Buffer.from(jwk.e, 'base64url').toString('hex'));
const fallbackMatches = BigInt('0x' + fallback.n_hex.replace(/^0x/i, '')) === publicModulus &&
  BigInt(fallback.e) === publicExponent;
const fixture = {
  source_sha256: sourceHash,
  provenance: 'B 服更新主脚本 data_pack；固定随机数、合成账号与时间；无真实会话',
  key_hex: testKey.toString('hex'),
  cases,
  server: { command: 101, sequence: 7, message: reply, nonce_hex: nonce.toString('hex'), frame_hex: frame.toString('hex') },
};
const destination = path.join(directory, 'wire-vectors.json');
const encoded = JSON.stringify(fixture, null, 2) + '\n';
if (fs.existsSync(destination)) assert.equal(fs.readFileSync(destination, 'utf8'), encoded);
else fs.writeFileSync(destination, encoded, { flag: 'wx' });
assert.equal(fs.readFileSync(path.join(project, 'internal/protocol/testdata/bili_wire_vectors.json'), 'utf8'),
  encoded, 'Go 固定样本必须与原客户端重新生成的帧一致');
const evidence = { status: 'verified', client_frames: cases.length, server_frame: true,
  tampered_tag_rejected: true, public_key_bits: publicKey.asymmetricKeyDetails.modulusLength,
  public_key_spki_sha256: fingerprint, local_gate_key_matches: localMatches,
  fallback_public_key_matches: fallbackMatches,
  client_remainder_bytes: unpacked.remainData.byteLength };
const report = path.join(directory, 'wire-check.json');
const reportJSON = JSON.stringify(evidence, null, 2) + '\n';
if (fs.existsSync(report)) assert.equal(fs.readFileSync(report, 'utf8'), reportJSON);
else fs.writeFileSync(report, reportJSON, { flag: 'wx' });
console.log(JSON.stringify(evidence));
