# 多平台内容发布工具

## 项目简介

本项目用于帮助内容创作者将一份原始内容自动适配到公众号、知乎、B站、小红书等多个平台，并支持一键模拟发布，提升多平台分发效率。

## 当前状态

项目已实现以下核心能力：

- 内容输入与编辑
- 多平台格式适配
- 平台发布预览
- 一键模拟发布
- 用户注册与登录
- 后端 session 权限控制
- 发布任务队列
- 发布记录持久化
- 可扩展的平台适配器架构

## Demo 视频

https://drive.google.com/file/d/14swdJpbfiTN2wepqrnvS_ON4uaerzEBa/view?usp=vids_web

## 运行方式

### Go 后端模式

安装 Go 1.22 或更高版本后，在项目根目录运行：

```bash
go run ./server
```

然后在浏览器访问：

```text
http://localhost:8080
```

首次使用时，在页面顶部注册账号。第一个注册用户会自动成为 `admin`，后续用户默认为 `creator`。

### 静态页面模式

也可以直接打开 `index.html` 体验前端页面。此模式下页面会使用前端本地兜底逻辑完成内容适配和模拟发布。

## 依赖说明

本项目当前未引入第三方框架或第三方库。

- 前端：原生 HTML、CSS、JavaScript
- 后端：Go 标准库 `net/http`、`encoding/json`

## 原创功能说明

本项目后续实现的核心功能，包括内容编辑、多平台格式适配、模拟发布流程和平台扩展架构，均为本项目自主开发。

当前原创实现部分包括：

- 内容输入编辑器
- 公众号、知乎、B站、小红书平台适配规则
- 用户注册、登录、退出和 session 校验
- 基于 JSON 文件的本地持久化存储
- 异步发布任务队列
- 发布历史查询
- Go 后端适配 API：`POST /api/adapt`
- Go 后端模拟发布 API：`POST /api/publish/mock`
- 前端多平台预览和模拟发布交互

## 后端接口

- `POST /api/auth/register`：注册用户
- `POST /api/auth/login`：登录用户
- `POST /api/auth/logout`：退出登录
- `GET /api/me`：获取当前登录用户
- `POST /api/adapt`：生成多平台适配内容
- `POST /api/publish/mock`：创建异步模拟发布任务
- `GET /api/publish/jobs`：查询发布历史
- `GET /api/publish/jobs?id=任务ID`：查询单个发布任务

## 模拟平台发布说明
在没有平台授权凭证的情况下，本项目使用 mock publish connector 模拟发布结果；后续接入真实平台时，可以在后端为每个平台新增独立 connector，实现授权、素材上传、发布提交和状态回查。

