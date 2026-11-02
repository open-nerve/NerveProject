# M5 / P2 阶段评审记录

> 实施中：T1完成，T2–T14待执行；没有进行整分支终审、没有合并。基线main85dc4087，预检提交72a3e58b，见[re-34](../../../reviews/re-34-M5P2预检.md)。原件在本工作树 `.superpowers/sdd/P2-assets`。

## 1. 方式

主控按executing-plans逐Task实施/TDD/自查，阶段末deep-review三视角与一次新鲜整分支审查。P2完整builder check仅为设计预检，不当产品审查。

## 2. 验收进度

A1–A12均保持未完成。T1提供用途/范围/父组合、metadata≤3候选/16KiB及受限图片校验；它不提供API、数据库或完整恢复链，不能据此勾选阶段标准。

## 3. T1已完成的验证与裁定

- domain/content 的setup编译红单列；最小零行为声明后真实性质红，最终绿。四格式实物头、完整字节hash、同长度异尾、头部≤1MiB、边长/像素限制、过短/过长流、取消均覆盖。1×1 WebP夹具另用官方完整解码确认可读。
- `task-1-mutations.json`记录4个行为变异被杀：取消候选上限、只摘要首块、只信声明图片、遗漏EOF取消。原字节hash恢复核验。
- `task-1-affected-green.log`：asset全包/archtest/config/httpserver的race通过。`task-1-lint.log`：资产相关golangci-lint 0 issues。关键词76规则/14例外无命中。
- `task-1-full-go.log/json`：实际make test完整server/tools通过，退出0；测试输入逐文件hash在task-1-tested-inputs.json，原件没有假记全项目七项已跑。
- x/image0.47的必要传递链为x/sys0.49、x/text0.43，后者要求现有x/tools0.51/x/sync0.24。go mod tidy把原本直接import的rivertype归到直接依赖，版本未变。模块图保存在task-1-dependency-graph.txt。
- validator只消费io.Reader，Close由拥有ObjectReader的app在T7验证；T1不声称证明上层Close。ETag最长1024字节且总metadata≤16KiB，超长显式拒绝不截断，P3真实provider合同继续核对。
- P1 inline类型检查复用同一RasterMediaType，既有签名读回归通过，无新类型放行。图片仅头部验证，不冒称全像素/动画安全证明。

## 4. 整分支审查

未开始；T13执行。

## 5. 与plan差异

T1确认必须的完整Go依赖传递链，补齐spec/plan；T1无ETag输入，原“ETag当摘要”变异具体化为漏算尾部字节，真正ETag版本与SHA不同职责在T7封存组合验证。每条局部裁定及代价另见台账。

## 6. 后续责任

T2开始有限清单和临时残留清理。P4/P5绑定、P3 S3、P6/P7页面以及P1遗留C12 Minor都未完成；输入handoff保持open。

## 7. 已知限制

当前仅基础纯规则/校验；尚无资产业务路由、file_assets或新清理worker。不能根据T1通过声称F1/F3/F13/F14/F15已满足。
