"""只读扫描已解包 B224、原型及 Go 注册，输出协议定位清单，不复制客户端代码"""

import argparse
import ast
import bisect
import hashlib
import re
from pathlib import Path

from python_baseline import SOURCE
from repair_bili224_save import load_tables

PROJECT = Path(__file__).resolve().parents[1]
CLIENT = PROJECT / "data/assets/bili/analysis/local-client/base224-local/original-index.js"
CLIENT_HASH = "89513c054d788a55617a86b4820578019b0135e5f3aeb86692bb76b1af39d5c6"
PYTHON_HASH = "13ef3e5dfcda8ed996693711c5c573df533b41b89d812d49ce4a132fc49bda25"


def arguments(source, start):
    """读取 request 的前两个参数，保留嵌套括号并跳过字符串与注释"""
    result, stack, begin, index = [], [], start, start
    pairs = {"(": ")", "[": "]", "{": "}"}
    while index < len(source):
        char = source[index]
        if char in "\"'`":
            quote = char
            index += 1
            while index < len(source) and source[index] != quote:
                index += 2 if source[index] == "\\" else 1
            if index == len(source):
                raise ValueError("未闭合字符串")
        elif source.startswith("//", index):
            end = source.find("\n", index)
            index = len(source) if end < 0 else end
            continue
        elif source.startswith("/*", index):
            end = source.find("*/", index + 2)
            if end < 0:
                raise ValueError("未闭合注释")
            index = end + 2
            continue
        elif char in pairs:
            stack.append(pairs[char])
        elif char in ")]}" and stack:
            if stack.pop() != char:
                raise ValueError("括号不匹配")
        elif char == ")" and not stack:
            result.append(source[begin:index].strip())
            return result
        elif char == "," and not stack:
            result.append(source[begin:index].strip())
            if len(result) == 2:
                return result
            begin = index + 1
        index += 1
    raise ValueError("未闭合 request")


def python_branches(source):
    tree = ast.parse(source)
    groups = {}
    for node in tree.body:
        if isinstance(node, ast.Assign):
            try:
                value = ast.literal_eval(node.value)
            except (ValueError, TypeError):
                continue
            for target in node.targets:
                if isinstance(target, ast.Name) and isinstance(value, (set, list, tuple)):
                    groups[target.id] = value
    session = next(node for node in tree.body if isinstance(node, ast.ClassDef) and node.name == "Session")
    dispatch = next(node for node in session.body if isinstance(node, ast.FunctionDef) and node.name == "dispatch")
    found = {}
    for node in ast.walk(dispatch):
        if not isinstance(node, ast.If):
            continue
        if not any(isinstance(term, ast.Name) and term.id == "name" for term in ast.walk(node.test)):
            continue
        names = [term.value for term in ast.walk(node.test)
                 if isinstance(term, ast.Constant) and isinstance(term.value, str) and term.value.startswith("MSG_")]
        for term in ast.walk(node.test):
            if isinstance(term, ast.Name):
                names.extend(value for value in groups.get(term.id, ()) if isinstance(value, str) and value.startswith("MSG_"))
        for name in names:
            found.setdefault(name, node.lineno)
    return found


def go_requests(source):
    constants = {name: int(number) for name, number in re.findall(r"\b(Command\w+)\s+(?:uint16\s*)?=\s*(\d+)", source)}
    body = source.split("func CommandName(", 1)[1]
    found = {}
    for cases, name in re.findall(r'case\s+([^:]+):\s*return\s+"(MSG_[A-Z0-9_]+)",\s*true', body):
        for case in cases.split(","):
            found[name] = constants[case.strip()]
    return found


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--output", type=Path, default=PROJECT / "docs/bili224-command-coverage.md")
    args = parser.parse_args()
    client_bytes, python_bytes = CLIENT.read_bytes(), SOURCE.read_bytes()
    if hashlib.sha256(client_bytes).hexdigest() != CLIENT_HASH or hashlib.sha256(python_bytes).hexdigest() != PYTHON_HASH:
        raise ValueError("证据源散列变化，需先重新核对扫描规则")
    source = client_bytes.decode("utf-8")
    modules = [(match.start(), match.group(1)) for match in re.finditer(r"^(\w+): \[ function\(", source, re.M)]
    methods = [(match.start(), match.group(1)) for match in re.finditer(r"\b\w+\.prototype\.(\w+)\s*=\s*function\(", source)]
    module_offsets, method_offsets = [p for p, _ in modules], [p for p, _ in methods]
    protocol = load_tables(PROJECT, ("SERVER_Protocol",))["SERVER_Protocol"]
    mapping = {}
    for name, row in protocol.items():
        if name.startswith("MSG_"):
            if not isinstance(row, dict) or type(row.get("ProtocolCode")) is not int:
                raise ValueError(f"协议编号结构未知: {name}")
            mapping[name] = row["ProtocolCode"]
    branches = python_branches(python_bytes.decode("utf-8-sig"))
    supported = go_requests((PROJECT / "internal/game/commands.go").read_text(encoding="utf-8"))
    calls, unresolved = {}, []
    for match in re.finditer(r"\.request\s*\(", source):
        line = source.count("\n", 0, match.start()) + 1
        module_index = bisect.bisect_right(module_offsets, match.start()) - 1
        method_index = bisect.bisect_right(method_offsets, match.start()) - 1
        module = modules[module_index][1] if module_index >= 0 else "unknown"
        method = methods[method_index][1] if method_index >= 0 and methods[method_index][0] >= modules[module_index][0] else "unknown"
        location = f"{module}.{method}:{line}"
        values = arguments(source, match.end())
        name_match = re.fullmatch(r'[\"\'](MSG_[A-Z0-9_]+)[\"\']', values[1]) if len(values) >= 2 else None
        if not name_match:
            unresolved.append(location)
            continue
        calls.setdefault(name_match.group(1), []).append(location)
    output = ["# B224 客户端 request 静态覆盖清单", "",
              "状态：已验证静态调用点和注册状态；请求实际是否发生、字段语义及实机通过均不能由本表推定", "",
              "生成：`python -B scripts/scan_client_commands.py`，只读原脚本、配置和代码，不读取玩家存档", "",
              f"客户端 SHA-256：`{CLIENT_HASH}`；Python SHA-256：`{PYTHON_HASH}`", "",
              f"Go 受理 {len(supported)} 种请求；静态发现 {len(calls)} 个字面量命令、{sum(map(len, calls.values()))} 处调用；另有 {len(unresolved)} 处动态/其他 request 待人工核对", "",
              "只统计 `.request(请求, 字面量命令, ...)`；不覆盖直接底层发包、动态命令名或全部主动推送", "",
              "Go“已受理”只表示分发已接入，空态、固定原型模型、错误边界和业务范围须查实现文档", "",
              "| CMD | 名称 | 客户端 模块.方法:行 | Python 分支行 | Go 受理 |", "|---:|---|---|---:|---|"]
    missing = [name for name in calls if name not in supported]
    with_branch = [name for name in missing if name in mapping and name in branches]
    no_branch = [name for name in missing if name in mapping and name not in branches]
    no_number = [name for name in missing if name not in mapping]
    output[14:14] = [
        f"字面量未受理 {len(missing)} 个：编号已知且有 Python 显式分支 {len(with_branch)} 个、编号已知但未找到显式分支 {len(no_branch)} 个、未在 B224 协议表找到编号 {len(no_number)} 个", "",
        "“未找到显式分支”是静态扫描结果，仍须追踪动态分发和辅助函数；原型默认成功兜底不作为实现依据", "",
    ]
    for name in sorted(calls, key=lambda value: (int(mapping.get(value, 65536)), value)):
        number = mapping.get(name)
        status = "是" if name in supported else "否"
        if name in supported and number != supported[name]:
            status = "编号冲突，需核对"
        output.append(f"| {number if number is not None else '未知'} | {name} | {'<br>'.join(calls[name])} | {branches.get(name, '无显式分支')} | {status} |")
    output += ["", "## 动态调用及其他 request", "", "下列位置未从第二参数取到字面量消息名，必须人工追踪，不能推断已经覆盖", ""]
    output += [f"- `{item}`" for item in unresolved]
    output += ["", "## Go 已受理但本扫描未找到字面量调用", "", "常见原因是协议框架直接发包或动态命令名；不能据此删除实现", ""]
    output += [f"- `{number} {name}`" for name, number in sorted(supported.items(), key=lambda item: item[1]) if name not in calls]
    args.output.write_text("\n".join(output) + "\n", encoding="utf-8")
    print(f"Go requests={len(supported)}, literal commands={len(calls)}, calls={sum(map(len, calls.values()))}, unresolved={len(unresolved)}")
    print(f"Unsupported literals={len(missing)}, explicit Python={len(with_branch)}, no explicit branch={len(no_branch)}, unknown code={len(no_number)}")
    print(f"written: {args.output}")


if __name__ == "__main__":
    main()
