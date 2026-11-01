# 宿主回调认证文件插件

[English](README.md) | [简体中文](README_CN.md)

一个仅 Go 实现的示例插件：其 Management API 页面调用宿主的认证文件回调：

- `host.auth.list`
- `host.auth.get`
- `host.auth.get_runtime`
- `host.auth.save`

## 用途与范围

插件在 `/status` 注册了一个名为 `Host Auth Files` 的 Management API 资源，暴露路径为：

```text
/v0/resource/plugins/host-callback-auth-files/status
```

页面读取 URL 查询参数，调用宿主认证回调，并把结果渲染为 HTML。它不注册 executor、translator、auth provider、scheduler 能力。

## 构建

在本目录下执行：

```bash
cd go
go build -buildmode=c-shared -o host-callback-auth-files.dylib .
rm -f host-callback-auth-files.h
cp host-callback-auth-files.dylib ../../plugins/
```

按目标系统使用对应扩展名：macOS 用 `.dylib`，Linux 用 `.so`，Windows 用 `.dll`。

## 配置

把动态库放入所配置的插件目录（`plugins.dir`，默认 `plugins`），文件基名需与插件 ID 一致：`plugins/host-callback-auth-files.dylib` 对应 `plugins.configs.host-callback-auth-files`。

```yaml
plugins:
  enabled: true
  dir: "plugins"
  configs:
    host-callback-auth-files:
      enabled: true
      priority: 1
```

本插件没有自己的配置字段。

## 资源 URL 示例

列出全部认证文件：

```text
http://localhost:8080/v0/resource/plugins/host-callback-auth-files/status?op=list
```

按认证索引读取物理 JSON：

```text
http://localhost:8080/v0/resource/plugins/host-callback-auth-files/status?op=get&auth_index=<AUTH_INDEX>
```

按认证索引读取运行时信息：

```text
http://localhost:8080/v0/resource/plugins/host-callback-auth-files/status?op=runtime&auth_index=<AUTH_INDEX>
```

保存物理 JSON：

```text
http://localhost:8080/v0/resource/plugins/host-callback-auth-files/status?op=save&name=example-auth.json&json=%7B%22type%22%3A%22gemini%22%2C%22email%22%3A%22demo%40example.com%22%2C%22api_key%22%3A%22demo-key%22%7D
```

## 参数

| 参数 | 说明 |
| --- | --- |
| `op` | 取值为 `list`（默认）、`get`、`runtime`（别名 `get_runtime`）、`save`。 |
| `auth_index` | `get` 和 `runtime` 必填。 |
| `name` | `save` 必填。必须以 `.json` 结尾。 |
| `json` | `save` 必填。必须是合法 JSON。 |

`list`、`get`、`runtime`、`save` 之外的 op 值暂不支持，页面会直接返回错误。

## 说明

- `host.auth.get` 返回认证文件的物理 JSON。
- `host.auth.get_runtime` 返回运行时凭据元数据。
- `host.auth.save` 把 JSON 写入认证目录并更新运行时认证记录。
