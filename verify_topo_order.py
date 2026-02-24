#!/usr/bin/env python3
"""
验证 status.json 中的模块顺序是否符合 README.md 定义的拓扑序
"""

import json
import re
from collections import defaultdict, deque

# 从 README.md 解析出的模块依赖关系
# 基于模块依赖关系图 (第144-210行)
DEPENDENCIES = {
    # Phase 0: 环境准备 (最高优先级)
    "go_env_setup": [],  # 无依赖，必须先完成

    # Phase 1: 基础框架 (无依赖，可并行)
    "config": [],
    "git": [],
    "parser": [],
    "deploy": [],
    "errors": [],
    "prompts": [],
    "logging": ["config"],  # 依赖 Config
    "call_cli": ["config", "logging"],  # 依赖 Config, Logging

    # Phase 2: 核心层
    "state": ["config"],  # 依赖 Config
    "cli": ["config", "logging"],  # 依赖 Config, Logging
    "executor": ["state", "parser", "call_cli"],  # 依赖 State, Parser, Call CLI

    # Phase 3: 命令层
    "research_cmd": ["config", "logging", "call_cli"],
    "plan_cmd": ["config", "logging", "parser", "call_cli"],
    "doing_cmd": ["config", "logging", "state", "git", "parser", "call_cli"],  # README中还提到依赖Executor
    "stat_cmd": ["config", "logging", "state", "git", "parser"],
    "reset_cmd": ["config", "logging", "state", "git"],

    # Phase 4: 验证层
    "e2e_test": [],  # 依赖所有模块，但拓扑排序中放最后即可
}


def topological_sort(dependencies):
    """
    对模块进行拓扑排序
    返回排序后的模块列表和是否成功
    """
    # 构建入度表和邻接表
    in_degree = {module: 0 for module in dependencies}
    adj = defaultdict(list)

    for module, deps in dependencies.items():
        for dep in deps:
            if dep in dependencies:
                adj[dep].append(module)
                in_degree[module] += 1

    # Kahn算法
    queue = deque()
    for module, degree in in_degree.items():
        if degree == 0:
            queue.append(module)

    sorted_modules = []
    while queue:
        # 为了可预测性，按字母顺序处理入度为0的节点
        queue = deque(sorted(queue))
        module = queue.popleft()
        sorted_modules.append(module)

        for neighbor in adj[module]:
            in_degree[neighbor] -= 1
            if in_degree[neighbor] == 0:
                queue.append(neighbor)

    # 检查是否有环
    if len(sorted_modules) != len(dependencies):
        remaining = set(dependencies.keys()) - set(sorted_modules)
        return sorted_modules, False, f"存在循环依赖或无法解析的模块: {remaining}"

    return sorted_modules, True, "OK"


def get_module_order_from_status_json(filepath):
    """
    从 status.json 中读取模块顺序
    JSON对象的键顺序即为模块顺序
    """
    with open(filepath, 'r') as f:
        data = json.load(f)

    # 保持JSON中的顺序（Python 3.7+ dict保持插入顺序）
    modules = list(data.get("modules", {}).keys())
    return modules, data


def verify_order(actual_order, expected_order):
    """
    验证实际顺序是否符合拓扑序
    只要满足依赖关系即可，不要求完全一致
    """
    position = {name: idx for idx, name in enumerate(actual_order)}
    errors = []

    for module, deps in DEPENDENCIES.items():
        if module not in position:
            errors.append(f"模块 '{module}' 不在 status.json 中")
            continue

        for dep in deps:
            if dep not in position:
                errors.append(f"依赖 '{dep}' 不在 status.json 中")
                continue

            if position[dep] >= position[module]:
                errors.append(
                    f"模块 '{module}' (位置 {position[module]}) 应该在依赖 '{dep}' (位置 {position[dep]}) 之后"
                )

    return len(errors) == 0, errors


def main():
    status_file = "/home/sankuai/dolphinfs_sunquan20/ai_coding/Coding/morty/.morty/status.json"

    print("=" * 60)
    print("拓扑排序验证工具")
    print("=" * 60)

    # 1. 计算理论拓扑序
    print("\n[1] 根据 README.md 计算理论拓扑序:")
    topo_order, success, msg = topological_sort(DEPENDENCIES)

    if not success:
        print(f"  错误: {msg}")
        return

    print("  理论拓扑序 (Kahn算法):")
    for i, module in enumerate(topo_order, 1):
        deps = DEPENDENCIES.get(module, [])
        dep_str = f" (依赖: {', '.join(deps)})" if deps else " (无依赖)"
        print(f"    {i:2d}. {module:15s}{dep_str}")

    # 2. 读取 status.json 中的实际顺序
    print("\n[2] 读取 status.json 中的实际模块顺序:")
    actual_order, data = get_module_order_from_status_json(status_file)

    print(f"  文件: {status_file}")
    print(f"  模块总数: {len(actual_order)}")
    print("  实际顺序:")
    for i, module in enumerate(actual_order, 1):
        deps = DEPENDENCIES.get(module, [])
        dep_str = f" (依赖: {', '.join(deps)})" if deps else " (无依赖)"
        print(f"    {i:2d}. {module:15s}{dep_str}")

    # 3. 验证顺序
    print("\n[3] 验证实际顺序是否符合拓扑序:")
    is_valid, errors = verify_order(actual_order, topo_order)

    if is_valid:
        print("  ✅ 验证通过! status.json 中的模块顺序符合拓扑排序")

        # 额外检查：是否与理论顺序完全一致
        if actual_order == topo_order:
            print("  ✅ 顺序与理论拓扑序完全一致")
        else:
            print("  ℹ️  顺序符合拓扑序，但与理论顺序不完全一致（这是正常的，拓扑序可能有多个）")
    else:
        print("  ❌ 验证失败!")
        for error in errors:
            print(f"     - {error}")

    # 4. 验证 go_env_setup 是否排第一
    print("\n[4] 验证 go_env_setup 优先级:")
    if actual_order[0] == "go_env_setup":
        print("  ✅ go_env_setup 排在第一位，符合要求")
    else:
        print(f"  ❌ go_env_setup 不是第一位，当前第一位是: {actual_order[0]}")

    # 5. 验证 e2e_test 是否排最后
    print("\n[5] 验证 e2e_test 位置:")
    if actual_order[-1] == "e2e_test":
        print("  ✅ e2e_test 排在最后一位，符合要求")
    else:
        print(f"  ℹ️  e2e_test 不是最后一位，当前位置: {actual_order.index('e2e_test') + 1}")

    # 6. 验证关键依赖关系
    print("\n[6] 验证关键依赖关系:")
    checks = [
        ("logging", "config", "logging 应该在 config 之后"),
        ("call_cli", "logging", "call_cli 应该在 logging 之后"),
        ("state", "config", "state 应该在 config 之后"),
        ("cli", "logging", "cli 应该在 logging 之后"),
        ("executor", "state", "executor 应该在 state 之后"),
        ("executor", "call_cli", "executor 应该在 call_cli 之后"),
        ("doing_cmd", "state", "doing_cmd 应该在 state 之后"),
        ("doing_cmd", "git", "doing_cmd 应该在 git 之后"),
        ("stat_cmd", "state", "stat_cmd 应该在 state 之后"),
        ("reset_cmd", "state", "reset_cmd 应该在 state 之后"),
    ]

    all_passed = True
    for module, dep, desc in checks:
        if module in actual_order and dep in actual_order:
            if actual_order.index(module) > actual_order.index(dep):
                print(f"  ✅ {desc}")
            else:
                print(f"  ❌ {desc}")
                all_passed = False

    print("\n" + "=" * 60)
    if is_valid and all_passed:
        print("总结: ✅ 所有验证通过!")
    else:
        print("总结: ❌ 存在验证失败项，请检查")
    print("=" * 60)


if __name__ == "__main__":
    main()
