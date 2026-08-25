# gls

`gls` 是一个面向 GitLab 的轻量级代码检索命令行工具。它可以在单个项目或整个群组（包括子群组）的仓库中，按关键字和分支并发搜索代码，并将命中内容直接显示或导出。

适用于授权范围内的代码审计、配置排查、迁移前检查与日常开发定位。搜索和导出结果可能含有敏感信息，请妥善保管。

## 特性

- 群组和项目两种搜索范围，群组搜索自动包含子群组
- 多关键字、多分支搜索；默认分支为 `master` 与 `main`
- GitLab API 自动分页，遇到限流或服务端错误会重试
- 可调并发数，适合较大的群组
- 输出到终端、CSV、JSON 或 XLSX

## 安装

需要 Go 1.22 或更高版本：

```bash
go build -o gls .
```

## 快速开始

在群组内搜索多个关键字，并导出为 Excel：

```bash
./gls -u https://gitlab.example.com -t YOUR_TOKEN -g 123 \
  -k password -k api_key -o audit.xlsx
```

仅搜索一个项目的指定分支：

```bash
./gls -u gitlab.example.com -t YOUR_TOKEN -p 456 \
  -b feature/login -k redis://,mysql:// -o results.json
```

不指定 `-o` 时，结果会输出到终端。

## 参数

| 参数 | 说明 |
| --- | --- |
| `-u`, `--url` | GitLab 地址；省略协议时默认使用 HTTPS |
| `-t`, `--token` | GitLab Private Token |
| `-k`, `--keywords` | 搜索关键字；可多次传入或以逗号分隔 |
| `-g`, `--group` | 搜索的群组 ID |
| `-p`, `--project` | 搜索的项目 ID |
| `-b`, `--branch` | 搜索分支；可多次传入或以逗号分隔 |
| `-o`, `--output` | 输出路径，支持 `.xlsx`、`.csv`、`.json` |
| `--workers` | 群组搜索并发数，默认 `10` |
| `-v`, `--verbose` | 输出详细执行日志 |

`--group` 和 `--project` 必须且只能指定一个。Token 需要具备目标资源的读取权限，通常使用 `read_api` 或 `api` 权限即可。
