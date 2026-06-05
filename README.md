# ReadBooks

本地漫画阅读桌面应用。

## 技术栈

- **后端**：Go 1.25 + Wails v3 + modernc.org/sqlite
- **前端**：Vue 3 + Vite + Js
- **桌面框架**：Wails v3

## 功能

- 默认漫画目录设置 （根据名字自动排除已有漫画）
- 漫画文件夹导入（单文件夹 / 批量扫描）
- 封面缩略图生成（400px 宽，JPEG 压缩，内存缓存）
- 阅读进度记录（localStorage + 后端双写）
- 横屏 / 竖屏阅读模式
- 标签管理（创建、编辑、删除、搜索）
- 漫画批量删除（含源文件）
- 暗色 / 亮色主题切换

## 目录结构

```
ReadBooks/
  main.go             
  appservice.go       
  comicutil.go        漫画扫描、元数据解析（JM下载器下载的漫画扫描）
  internal/storage/   SQLite
  frontend/           
    src/views/        
    src/components/   
  build/              
```


## 构建

```bash

wails3 dev     

wails3 build   // 生产版本 .exe
```

## 数据存储

- **数据库**：SQLite（`readbooks.db`）
- **配置**：`state.json`（窗口尺寸、默认漫画目录）


## 图片服务

封面通过 Wails AssetOptions Middleware 提供：

```
GET /api/image?cover=1&p={filePath}
```

- `cover=1`：返回压缩缩略图（宽 400px，JPEG quality 80）
- 无 cover 参数：返回原图
- 内存缓存最多 20 张缩略图

## 依赖

| 包 | 用途 |
|---|---|
| github.com/wailsapp/wails/v3 | 桌面应用框架 |
| modernc.org/sqlite | SQLite 驱动（纯 Go） |
| github.com/sqweek/dialog | 系统文件夹选择对话框 |
