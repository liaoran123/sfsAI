# sfsAI

> **AI 原生时代的边缘数据底座 — 小而美**

sfsAI 是面向 AI Agent 的轻量级记忆体 Sidecar。它不做通用数据库，只做三件事：**记忆存取、隐私安全、就近服务**，每一件都做到极致。

基于 [sfsDb](https://github.com/liaoran123/sfsDb) LSM-Tree 引擎，提供 **8MB 二进制 + 50MB 常驻内存** 的极致轻量级 AI 记忆体服务。

---

## 快速开始

```bash
# 启动 Sidecar 服务
go run ./cmd/sfsai

# 自定义数据目录和端口
go run ./cmd/sfsai -db /data/my-device -addr :9630
```

启动后，AI Agent 即可通过 `localhost:8630` 的 HTTP API 存取记忆。

---

## 架构

```
AI Agent (任何语言)
      │ HTTP/JSON
      ▼
┌─────────────┐   Sidecar 进程，伴生运行
│   sfsAI     │   ~8MB 二进制
│  (Sidecar)  │   聚焦 AI 记忆 API
└──────┬──────┘
       │ import
       ▼
┌─────────────┐
│   sfsDb     │   ~4MB 嵌入式引擎
│  (Engine)   │   LSM-Tree、索引、事务、加密
└─────────────┘
```

详细架构说明见 [项目架构.md](文档/项目架构.md)

---

## HTTP API

| 方法 | 路径 | 功能 |
|------|------|------|
| `POST` | `/api/v1/memories` | 写入记忆 |
| `GET` | `/api/v1/memories` | 搜索记忆 |
| `GET` | `/api/v1/memories/:sid/:mid` | 单条查询 |
| `GET` | `/api/v1/memories/recent/:sid` | 最近记忆 |
| `DELETE` | `/api/v1/memories/wipe` | 擦除记忆（被遗忘权） |
| `POST` | `/api/v1/memories/compress/:sid` | 压缩蒸馏 |
| `POST` | `/api/v1/memories/semantic` | 语义搜索 |
| `GET` | `/api/v1/health` | 健康检查 |
| `GET` | `/api/v1/stats` | 状态统计 |

---

## SDK 使用

```go
import "sfsAI/pkg/sdk"

client := sdk.NewClient("http://localhost:8630")

// 写入记忆
result, _ := client.InsertMemory("session-1", "用户打开了智能家居面板", nil, nil)

// 语义搜索
results, _ := client.SemanticSearch("session-1", []float32{0.1, 0.2, 0.3}, 5)
```

---

## 核心特性

- **MemoryUnit** — `memory_id + session_id + content + embedding + metadata` 复合数据结构
- **Conversation Stream** — `(session_id, created_at)` 复合索引，毫秒级时序扫描
- **Semantic Search** — 余弦相似度向量检索
- **Memory Compression** — 自动蒸馏过期记忆为摘要
- **Right to be Forgotten** — 物理删除，配合密钥销毁实现真正遗忘
- **Zero-Trust 加密** — AES-256-GCM 透明加密，设备绑定密钥，默认启用

---

## 配置

支持命令行参数和配置文件两种方式：

```bash
go run ./cmd/sfsai -h

# -db     数据库目录（默认 ./sfsai_data）
# -addr   HTTP 监听地址（默认 :8630）
# -config 配置文件路径（可选）
```

---

## 构建

```bash
# 构建二进制
go build -o sfsai ./cmd/sfsai

# 交叉编译（ARM 边缘设备）
GOOS=linux GOARCH=arm64 go build -o sfsai-arm64 ./cmd/sfsai
```

---

## 开源协议

MIT License — 详见 [LICENSE](LICENSE)

---

## 商业合作

sfsAI 采用 **开源免费 + 商业服务** 模式：

| 服务类型 | 说明 | 适合场景 |
|---------|------|---------|
| **开源社区版** | MIT 协议，免费使用 | 个人开发者、原型验证 |
| **商业 OEM 授权** | License 授权，批量预装 | 硬件方案商、IDH、量产厂商 |
| **行业定制服务** | 按需定制功能 | 军工、电力、医疗、特种检测设备 |
| **技术支持** | 7x24 支持服务 | 企业级部署 |

### 授权定价参考

| 授权类型 | 适用规模 | 单价（参考） |
|---------|---------|-------------|
| 小批量授权 | < 1000台 | 50元/台 |
| 中批量授权 | 1000~10000台 | 30元/台 |
| 大批量授权 | > 10000台 | 10~20元/台 |
| 定制开发 | 按需报价 | 1500~3000元/人天 |

### 合作权益

- 免费提供测试包、技术 Demo、部署教程
- 样机免费商用试用，无试用门槛
- 技术对接、移植适配、BUG 快速修复
- 优先获取新版本、新功能、行业定制权限

详细合作条款见 [PARTNERSHIP.md](PARTNERSHIP.md)

### 联系方式

欢迎工业网关、AI 硬件、嵌入式方案商、系统集成商洽谈合作：
- 微信：[预留联系方式]（备注：sfsAI 合作）
- 邮箱：sfsweb@qq.com