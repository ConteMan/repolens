# M6 画板清单

| 画板 | 视口 | 主任务 | 必备状态 |
|---|---:|---|---|
| Foundations | 1440 | 语义颜色、间距、字体、控件规格 | 正常、warning、blocking、success |
| Components | 1440 / 390 | Button/Input/Select/Toggle/Badge/Alert/Dialog/Skeleton | normal、focus、disabled、error |
| 项目选择 | 1440 / 390 | 输入本地 Git 工作树绝对路径 | 加载、空、非 Git、不可读 |
| 配置编辑 | 1440 / 390 | 编辑真实仓库域字段及 rules | 默认、空配置、YAML 错误、字段错误 |
| 写入确认 | 1440 / 390 | 理解并确认 diff | 无变更、冲突、写入失败、未知字段影响 |
| 构建结果 | 1440 / 390 | 查阅进度与构建产物 | 进行中、成功含 warning、失败 |

Pencil 恢复后，唯一 `prototypes/repolens-ui.pen` 按 `00 Foundations`、`01 Components`、`02 Project`、`03 Config`、`04 Write and Build`、`05 Narrow` 建立。每个业务组件使用 reusable component/ref；PNG 命名为 `<画板 ID>-<Pencil 节点 ID>.png` 并回填评审记录。
