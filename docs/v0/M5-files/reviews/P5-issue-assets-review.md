# M5/P5 阶段评审记录

> T1完成：有序图片内容身份、规范UUID提取、内容/域完整race和调用方/架构验证、四项行为变异及lint/关键词守卫通过；T2待实施。分支codex/m5-p5，基线main 0f93100a05304d26b074790633b5659285813156；P4精确main CI41641099230全绿并完成归档清理。

## 1. 评审方式

三份spec经一次fresh builder check，H1按原准备件隐私合同修正；主控做15任务plan的覆盖、接口、路径、步骤与Review Focus自查。记录见[re-40](../../../reviews/re-40-M5P5预检.md)。逐Task内联TDD，最终一次fresh deep-review及主控复核；当前只有T1实施证据，尚无阶段最终整审结论。

## 2. 验收标准

主spec A1–A9均未完成。逐项实际证据随任务记录，不能用预检代替测试或准确分支/main CI。

## 3. 执行中的决定

预检局部决定、原始问题和代价见re-40 §2–§3；核心是不自动公开从未引用的草稿准备图片、批量目录保持真实计数、评论分页有界和JSON草稿兼容。无未裁定重要问题。

## 4. 整分支审查与处理

待T14，尚未执行。

## 5. 与plan的差异

T1新增两份image_identity_test.go承载身份边界，避免继续扩充既有混合测试；内容清洗与domain纯值各在自己的包验证。其余按plan执行。

## 6. 移交事项

M4-closeout M5-01/02/03在本段承担API/DB部分，P6/P7页面与P8终验继续各自负责。输入handoff仍open，CLI原文和旧copy监听风险不丢失，不提前关闭M5整体标准。

## 7. 已知限制

当前仅T1内容身份已实现；API和页面的阶段边界、未绑定旧附件的首次绑定、计数目录容量与短期URL bearer限制见spec。执行中遇到无法满足的架构/容量事实先保存并提交证据再上报。


## 8. T1有序图片身份与规范引用

既有Sanitizer→CompareDescription的8个真实断言RED证明换图、换序、重复数量、空位置及安全远程源变化会被skip_activity漏判（1.398秒）。随后Images有序投影与Equal修正；规范UUID提取先以空实现执行新断言取得独立RED（2.442秒），最终实现只解析规范小写连字符UUID，保留顺序与重复，不修改源值。零UUID仍作为规范候选交宿主绑定层拒绝，不能悄悄当成外部图片绕过校验。

image-component缺src记空字符串，raw img延续HasImage/既有清洗而不成为内部引用通道；安全远程源继续参与身份，未取得资产授权。domain不解析HTML，HTMLSanitizer签名不变。对应资料/附件绑定和真正API动态/版本证明仍在T2–T7，T1不提前关闭A1或整个阶段。

相同聚焦GREEN1.872秒，恢复后最终GREEN1.192秒；两完整包race3.073秒，完整issue/app与archtest race11.088秒；lint-go27.576秒，0issues；关键词76规则/14例外无命中，只有phase改为M5/P5，例外与预检基线完全相同。全部最终4331个非文档输入hash一致。没有新增API字段或生成源变更，本任务不重复完整七项或不受影响的容器套件，T13仍完整执行。

四项定向变异分别去掉图片Equal、对src排序、对src去重、允许非规范UUID；每项都命中具体行为断言，非编译失败，逐文件SHA恢复后再次GREEN。原件为本工作树`.superpowers/sdd/P5-issue-assets/task-1-*.log`、inputs/receipt、mutations.json；T1全部测试不启动容器。只读Plane的content_validator与issue/base对照及差异见plane-diff。
