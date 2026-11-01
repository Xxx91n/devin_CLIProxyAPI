# Scheduler 插件

[English](README.md) | [简体中文](README_CN.md)

这个插件用 Go 演示 CLIProxyAPI 的 C ABI 调度器能力。

它实现了：

- `plugin.register`
- `plugin.reconfigure`
- `scheduler.pick`

插件可以选择某个已配置的 auth ID、把路由决策委托给内置调度器，或者拒绝本次调度。

## 配置

在 `plugins.configs` 下添加插件配置：

```yaml
plugins:
  configs:
    scheduler:
      enabled: true
      priority: 1
      auth_id: ""
      delegate: ""
      deny: false
```

字段说明：

- `auth_id`：该 auth ID 出现在调度候选中时，选中它。
- `delegate`：把选择委托给内置调度器。支持的值是 `""`、`fill-first` 和 `round-robin`。
- `deny`：设为 `true` 时返回调度错误。

行为：

- `deny` 为 `true` 时，插件返回错误码为 `scheduler_denied` 的错误信封。
- `delegate` 为 `fill-first` 或 `round-robin` 时，插件返回 `DelegateBuiltin` 并把本次调度标记为已处理。
- `delegate` 为其他非空值时，插件不处理本次调度。
- `delegate` 为空且 `auth_id` 存在于候选中时，插件返回该 auth ID 并把本次调度标记为已处理。
- 没有规则匹配时，插件不处理本次调度。

## 局限

`deny` 优先于 `auth_id` 和 `delegate`。`deny` 为 `true` 时，另外两个字段不起作用。

## 构建

在本目录下：

```bash
cd go
go build -buildmode=c-shared -o /tmp/cliproxy-scheduler-plugin.so .
rm -f /tmp/cliproxy-scheduler-plugin.h
```

输出文件名即插件 ID，请按 `plugins.configs` 中使用的配置键命名产物。
