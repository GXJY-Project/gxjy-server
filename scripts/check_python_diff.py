"""使用发布包真实表运行差分，密钥只通过子进程环境传递"""
import os
from pathlib import Path
import subprocess
import sys

import python_baseline as baseline


def main():
    root = Path(__file__).resolve().parents[1]
    fixture = root / "data/assets/diff/real_baseline.json"
    subprocess.run([sys.executable, "-B", str(root / "scripts/python_baseline.py"), "--real-assets", "--output", str(fixture)], check=True, cwd=root)
    sys.dont_write_bytecode = True
    sys.path.insert(0, str(baseline.SOURCE.parent.parent))
    import gxjy_decrypt
    env = os.environ.copy()
    key = gxjy_decrypt.KEY
    env["GXJY_ASSET_KEY"] = key.decode() if isinstance(key, bytes) else key
    env["GXJY_DIFF_FIXTURE"] = str(fixture)
    asset_root = baseline.SOURCE.parent.parent / "apk/assets/assets"
    env["GXJY_DIFF_ASSET_ROOT"] = str(asset_root)
    env["GXJY_DIFF_BUNDLE"] = str(asset_root / "resources/config.json")
    subprocess.run(["go", "test", "./internal/game", "./internal/assets", "./internal/server", "-run", "TestExistingPathsMatchPython|TestActualTablesMatchPython|TestHTTPMatchesPython|TestWebSocketMatchesPython", "-count=1"], check=True, cwd=root, env=env)


if __name__ == "__main__":
    main()
