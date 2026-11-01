# Request Lifecycle 插件

[English](README.md) | [简体中文](README_CN.md)

这个 Go 动态库插件演示请求准入、主动终止和请求终止生命周期处理。RPC schema 版本低于 2 的宿主暂不支持。

插件声明了两个可选能力：

- `request_interceptor`：在 `request.intercept_before` 中获取并发槽位，可以在任何上游执行器运行之前终止请求。
- `request_lifecycle_plugin`：在 `request.complete` 中释放槽位，覆盖成功、失败、被拒绝和已取消的请求。

宿主向请求拦截、响应拦截、流拦截以及终止事件 `RequestCompletion` 传入同一个 `RequestID`。

## 行为

- 最多允许 `max_concurrency` 个请求同时在途。
- 达到上限时返回自定义 `429` JSON 响应，带 `Retry-After: 1`。
- 原始请求体包含 `reject_keyword` 时返回自定义 `403` JSON 响应。
- 被终止的请求不会发往上游模型。
- 只释放此前已准入的请求 ID，因此被拒绝的请求和重复的终止事件不会把计数器减成负数。

## 配置

```yaml
plugins:
  enabled: true
  configs:
    request-lifecycle:
      enabled: true
      priority: 100
      max_concurrency: 2
      reject_keyword: "blocked"
```

把 `reject_keyword` 设为空字符串即可关闭关键字拒绝。

## 构建

在仓库根目录、macOS 下：

```bash
mkdir -p plugins/darwin/$(go env GOARCH)
go build -buildmode=c-shared \
  -o plugins/darwin/$(go env GOARCH)/request-lifecycle.dylib \
  ./examples/plugin/request-lifecycle/go
rm -f plugins/darwin/$(go env GOARCH)/request-lifecycle.h
```

Linux 或 FreeBSD 使用 `.so`，Windows 使用 `.dll`。

输出文件名即插件 ID，因此要配合上面的配置，示例产物必须命名为 `request-lifecycle`。

## 相关 RPC 方法

```text
plugin.register
plugin.reconfigure
request.intercept_before
request.intercept_after
request.complete
```

`request.complete` 是观察性回调。宿主异步调度它，被阻塞的插件不会拖慢响应投递；回调错误会被记录日志；所用上下文与下游取消分离，已取消的请求仍能释放自己的槽位。
