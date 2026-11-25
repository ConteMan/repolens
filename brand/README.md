# Repolens 品牌标志

[折页母版](repolens-logo.pen) 的 LG-01 画板包含书脊、书页和页角三条路径。正式主色为苔墨 #385442。CL-01 至 CL-03 引用这个组件。在 LG-01 中调整轮廓即可同步三个实例。每组配色的 mark、paper、light 变量分别用于标志、图标底色和深色页头上的标志。

从仓库根目录运行以下命令，需要 Python 3.9+ 和 ImageMagick 7。输出目录可自行指定，用新目录保存下一轮样张。

```sh
python3 brand/scripts/export.py /tmp/repolens-brand
```

脚本读取母版的路径、位置、尺寸和配色变量，导出透明 SVG、深色背景版、单色版、16 px 简化 favicon、32 px PNG，以及 128 和 1024 px 的桌面图标样张。LG-02 派生小图标保留弧形页线。网页在 16 CSS px 直接显示 SVG；位图使用 16 和 32 px 两档。母版使用未旋转的纯色路径，路径几何可直接编辑。

[首轮造型](rounds/R01-2026-11-25/index.html) · [第二轮配色](rounds/R02-2026-11-25/index.html)

正式资产位于 [assets/moss](assets/moss)。从仓库根目录运行 `python3 brand/scripts/export.py brand/assets --palette moss` 可重新生成。中英文 README 使用浅色和深色标志，MyGo 从正式资产生成 macOS 应用图标。[定稿记录](rounds/FINAL-2026-11-25.md)
