// 仅执行授权 B 服快照的属性和数据消费函数，入口、网络及界面依赖不执行
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const crypto = require('node:crypto');
const assert = require('node:assert/strict');
const { spawnSync } = require('node:child_process');

const [sourcePath, fixturePath, outputPath] = process.argv.slice(2);
assert.ok(sourcePath && fixturePath && outputPath, '需要脚本、表样本和独立输出路径');
const project = path.resolve(__dirname, '..');
assert.equal(spawnSync('git', ['check-ignore', '-q', outputPath], { cwd: project }).status, 0);
const source = fs.readFileSync(sourcePath, 'utf8');
const hash = crypto.createHash('sha256').update(source).digest('hex');
assert.equal(hash, 'a8a75655aafdf51a7b0cc397e64b9d3efe789367311cc9ae80b69a4b9901cda0');
const tables = JSON.parse(fs.readFileSync(fixturePath, 'utf8')).tables;
const reader = {
  requireRecordById(name, id) {
    assert.ok(tables[name] && Object.hasOwn(tables[name], id), '缺少 B 配置引用 ' + name + '/' + id);
    return tables[name][id];
  },
  requireDataByNameIdAndKey(name, id, key) { return this.requireRecordById(name, id)[key]; },
};
const context = vm.createContext({ cc: {
  _RF: { push() {}, pop() {} }, _decorator: { ccclass: (value) => value, property() {} },
} });
function moduleExports(name, dependencies) {
  const marker = '\n' + name + ': [ function(';
  let start = source.indexOf(marker);
  assert.ok(start >= 0 && source.indexOf(marker, start + marker.length) < 0, name);
  start++;
  const rest = source.slice(start + name.length + 2);
  const next = /\n[\w$]+: \[ function\(/.exec(rest);
  assert.ok(next, name);
  const array = rest.slice(0, next.index).trim().replace(/,$/, '');
  const entry = vm.runInContext('(' + array + ')', context, { timeout: 1000 });
  const result = { exports: {} };
  entry[0]((key) => {
    if (Object.hasOwn(dependencies, key)) return dependencies[key];
    // 未覆盖的依赖可以导入，执行它的方法会显式失败，不伪造客户端行为
    return { default: new Proxy(function() { throw new Error('未隔离依赖 ' + key); }, {
      get() { throw new Error('未隔离依赖 ' + key); },
    }) };
  }, result, result.exports);
  return result.exports;
}
const deps = { '../../../framework/data/DataReader': { default: reader } };
const attributes = moduleExports('HeroAttribute', deps).default;
const Equip = moduleExports('Equip', { ...deps, './EquipStar': { default: function() {} } }).default;
const protocols = JSON.parse(fs.readFileSync(path.join(path.dirname(sourcePath), 'protocol_map.json'), 'utf8'));
const report = { source_sha256: hash, provenance: 'B281 原 HeroAttribute、Equip 方法；真实配置和合成进度，无游戏入口或联网',
  heroes: [], equipment: [], initial: {}, awaken_contract: 'Hero.synchronize 使用 awakenInfo.open/stages，getAwaken 读取配置数组' };
const routes = [];
function Model() {}
Model.prototype.addResponeHandler = (name) => routes.push(name);
const Bag = moduleExports('BagModel', { ...deps, '../../../framework/data/Model': { default: Model } }).default;
const bag = new Bag();
bag.initPush();
assert.ok(routes.includes('MSG_BAG_RES'));
report.bag_request_command = protocols.MSG_REQ_BAG;
report.bag_response_command = protocols.MSG_BAG_RES;
for (const [id, base] of Object.entries(tables.HeroBase)) {
  if (base.IsShow !== 1) continue;
  for (const [level, quality, star] of [[1, 0, 0], [7, 1, 1], [60, 3, 6]]) {
    const qualityId = `HeroQuality_${id}_${quality}`, starId = `HeroStar_${id}_${star}`;
    if (!tables.HeroQuality[qualityId] || !tables.HeroStar[starId]) continue;
    const hero = {
      getId: () => id,
      getBaseValueByName: (name) => base[name], getQualityId: () => qualityId,
      getStar: () => star, getLevel: () => level, getRarity: () => base.Rarity,
      getFirstCost: () => base.FirstCost, getCofigStar: () => base.Star, getConfigQuality: () => base.Quality,
    };
    const values = {};
    for (const name of ['Atk', 'PhyDef', 'MagDef', 'Hp']) {
      values[name] = attributes.calculationBaseAtt(hero, name);
      assert.ok(Number.isFinite(values[name]), id + '/' + name);
    }
    report.heroes.push({ id, level, quality, star, attrs: values });
  }
}
for (const [id, base] of Object.entries(tables.EquipBase)) {
  for (const [level, quality] of [[1, 0], [20, 0], [7, 1], [40, 1], [100, 7]]) {
    const qualityId = `EQE_${id}_${quality}`;
    const values = {};
    for (const name of ['Atk', 'PhyDef', 'MagDef', 'Hp']) {
      const value = attributes.calculationEquipAtt(name, id, qualityId, level, base.Rarity, null);
      assert.ok(Number.isFinite(value), id + '/' + name);
      if (value > 0) values[name] = value;
    }
    const equip = new Equip('synthetic');
    const levelId = reader.requireDataByNameIdAndKey('BaseConfig', 'Equip_Exp', 'Config')[base.Rarity]
      .replace(/1$/, String(level));
    equip.synchronize({ template_id: id, data: { level_id: levelId, quality: qualityId } });
    assert.equal(equip.getLevel(), level);
    assert.equal(equip.getStarNum(), base.InitStar);
    assert.equal(reader.requireDataByNameIdAndKey('EquipStar', equip.getStarId(), 'StarNum'), base.InitStar);
    assert.equal(equip.getEquipQualityId(), qualityId);
    report.equipment.push({ id, level, quality, star: base.InitStar, level_id: levelId,
      quality_id: qualityId, star_id: equip.getStarId(), attrs: values });
  }
}
for (const [key, row] of Object.entries(tables.BaseConfig)) {
  if (key.startsWith('Player_Init_')) report.initial[key] = row.Config;
}
// 记录实际消费字段，不能把旧 awaken_id 当作 B 服已接受的进度
const heroStart = source.indexOf('\nHero: [ function(') + 1;
const heroEnd = source.indexOf('\nHero', heroStart + 20);
const heroSource = source.slice(heroStart, heroEnd);
assert.ok(heroSource.includes('if (e.awakenInfo)') && heroSource.includes('return this.config.Awaken || [];'));
assert.ok(!heroSource.includes('awaken_id'));
const encoded = JSON.stringify(report, null, 2) + '\n';
if (fs.existsSync(outputPath)) assert.equal(fs.readFileSync(outputPath, 'utf8'), encoded);
else fs.writeFileSync(outputPath, encoded, { flag: 'wx' });
console.log(JSON.stringify({ client_hero_vectors: report.heroes.length, client_equipment_vectors: report.equipment.length,
  source_sha256: hash }));
const goReportPath = process.argv[5];
if (goReportPath) {
  const runtime = JSON.parse(fs.readFileSync(goReportPath, 'utf8'));
  const result = { status: 'verified', cases: [], initial_config_differences: [], limitations: [
    '仅检查 Equip 配置读取、Hero 基础字段与 awakenInfo；技能模块和事件发送使用隔离替身',
    '普通道具模板来自 33 表之外，本轮仅复核原型的数量与序列化行为',
    'Player_Init 配置差异需确认 B 服服务端下发策略，不能仅据客户端默认表推断完整新号流程',
  ] };
  // 这些替身仅接收同步通知，不提供养成、技能效果或 UI 的兼容结论
  function SyncModule() { this.synchronize = () => {}; }
  const Hero = moduleExports('Hero', { ...deps, './HeroStar': { default: SyncModule },
    './HeroQuality': { default: SyncModule }, '../../../framework/manager/EventManager': { default: { emit() {} } },
    '../../../GameConfig': {},
  }).default;
  for (const sample of runtime.cases) {
    const row = { name: sample.name, equipment_checked: 0, heroes_checked: 0, errors: [],
      star_level_config_differences: [], awaken_progress_ignored: [] };
    for (const [id, item] of Object.entries(sample.bag_data || {})) {
      if (item.item_type !== 'Equip') continue;
      const equip = new Equip(id);
      for (const [method, prepare] of [
        ['synchronize', () => equip.synchronize(item)], ['getLevel', () => equip.getLevel()],
        ['getStarNum', () => equip.getStarNum()], ['getLevelLimit', () => equip.getLevelLimit()],
        ['getSkillId', () => equip.getSkillId()],
      ]) {
        try { prepare(); }
        catch (error) { row.errors.push({ id, method, error: error.message }); }
      }
      row.equipment_checked++;
    }
    for (const [id, data] of Object.entries(sample.hero_data || {})) {
      try {
        const hero = new Hero(id);
        // 技能属于独立模块，本轮不以空技能替身宣布技能行为正确
        const input = { ...data };
        delete input.skills;
        hero.synchronize(input);
        assert.equal(hero.getQualityId(), data.base_attr.quality_id);
        assert.equal(hero.getDataId(), data.base_attr.hero_model.hero_surface);
        assert.deepEqual(Array.from(hero.getAwaken()), tables.HeroBase[id].Awaken);
        const configStar = reader.requireDataByNameIdAndKey('HeroStar', data.base_attr.star.star_id, 'Star');
        if (hero.getStar() !== configStar) row.star_level_config_differences.push({ id, client: hero.getStar(), config: configStar });
        if (tables.HeroBase[id].Awaken.length && (sample.hero_state[id] || {}).awaken > 0 && !hero.getAwakeIsOpen()) {
          row.awaken_progress_ignored.push(id);
        }
        row.heroes_checked++;
      } catch (error) { row.errors.push({ id, method: 'Hero.synchronize/getters', error: error.message }); }
    }
    if (row.errors.length || row.awaken_progress_ignored.length || row.star_level_config_differences.length) result.status = 'incompatible';
    result.cases.push(row);
    if (sample.initial_values) {
      const initial = sample.initial_values;
      for (const [field, key] of [['heroes', 'Player_Init_Hero'], ['head_image', 'Player_Init_Head'],
        ['wife', 'Player_Init_Wife'], ['currency', 'Player_Init_Resource']]) {
        if (JSON.stringify(initial[field]) !== JSON.stringify(report.initial[key])) {
          result.initial_config_differences.push({ field, go: initial[field] ?? null, bili_config: report.initial[key],
            status: '需确认服务端初始化策略' });
        }
      }
    }
  }
  const destination = path.join(path.dirname(goReportPath), 'client-consume-report.json');
  fs.writeFileSync(destination, JSON.stringify(result, null, 2) + '\n', { flag: 'wx' });
  console.log(JSON.stringify({ client_consume_status: result.status,
    equipment_read_errors: result.cases.reduce((sum, row) => sum + row.errors.length, 0),
    awaken_progress_ignored: result.cases.flatMap((row) => row.awaken_progress_ignored).length,
    star_level_config_differences: result.cases.flatMap((row) => row.star_level_config_differences).length,
    initial_config_differences: result.initial_config_differences.length }));
  process.exitCode = result.status === 'verified' ? 0 : 1;
}
