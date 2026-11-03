# M5/P5：附件分页、计数与排序子合同

> 通过预检的子合同；复用 issue 已有读取快照和完整目录传参，不直接读取 file_assets。

## 1. 消费方端口

issue/app/attachment_ports.go中声明：

```go
type Attachment struct {
    ID uuid.UUID
    Name, ContentType string
    Size int64
    CreatedAt time.Time
    CreatedByID *uuid.UUID
}
type AttachmentRead struct { Host ContentHost; After uuid.UUID; Limit int }
type AttachmentCount struct { IssueID uuid.UUID; Count int64 }
type AttachmentReader interface {
    ListAttachments(context.Context, AttachmentRead) ([]Attachment, error)
}
type AttachmentCounter interface {
    CountIssueAttachments(context.Context, []uuid.UUID) ([]AttachmentCount, error)
}
type AttachmentCountCatalog interface {
    ProjectAttachmentCounts(context.Context, []uuid.UUID, uuid.UUID, int) ([]AttachmentCount, error)
}
```

后三参为projectIDs、afterIssueID、pageSize；仅在issue已确认项目列表的同一只读快照调用，分页固定1000，不截断。asset自己的SQL只按用途ISSUE_ATTACHMENT、is_uploaded=true、bound_at非null、deleted_at=null及项目/issue范围聚合，不含描述/评论/draft/准备/墓碑；返回唯一非零issueID/count且升序。普通Count按返回ID分1000批，空IDs不查；缺项显式补0，DB/端口错误不能补0。

ListAttachments按host唯一的issue/comment/draft和对应普通附件用途过滤，固定UUID升序、limit+1 lookahead，验证已上传+已绑定+未删除。owner在同一read snapshot先授权，隐藏/不存在不先查资产；asset不能另开快照。在列表DTO不签URL、不读取磁盘、不修改行。

游标base64url JSON字段v=1,kind,id,after，严格拒绝额外字段、错误类型/非规范UUID/宿主错用400。请求limit沿shared.PageSize50/100。对一个已可见宿主，边界之外为空页；新插入/删除影响后续位置，不承诺跨请求冻结快照。

## 2. Issue所有出口

domain.Issue.AttachmentCount int64，API required/非负；集中在Locks.seen与Listing返回装配补批量值，create/readBack/草稿publish等绕过seen出口单独接相同批量端口。完整审计handler→app→response，含get/identifier/search/create/update/archive/unarchive/subitems/relations/member/list各形状，不只详情和列表。

统计授权不另复制HiddenFrom：ID来自已授权Issue集合；asset只统计它们自己的活附件。此字段不含隐藏工作项引用，不因asset上传人不同而少计。最大bigint比较和游标文本保持精度，不转float64。

## 3. 全局排序

普通排序：既有SQL产生分页结果后，所有实际返回Issue按去重ID一次批量补计数；group/subgroup所有返回cell合并，不能一页/一行一次查询。

详情、按编号读取和直接子项列表显式注入现有ReadTx，在首个宿主/目录读取前建立只读REPEATABLE READ；权限、工作项、子项计数、附件计数共用其context。原列表/关系/搜索沿已有ReadTx，写返回沿原写事务。所有投影计数只用已授权且实际输出的ID，不含lookahead；空输出不查。asset补全合法缺项0，issue消费端要求完整、唯一、非负且没有额外ID；端口失败或畸形结果整体失败，不伪装0。

attachment_count排序：在同一snapshot先通过ProjectAttachmentCounts取得所有授权项目的非零目录（每页1000、完整遍历），用成对UUID/bigint数组加入ListRead.Directories.AttachmentCounts；Flat/Groups/Subgroups三套SQL在filtered→source_keys阶段LEFT JOIN这个内存目录并coalesce0，之后再排序和游标分页。它与states/labels/assignees目录相同，不是跨模块表JOIN。全局排序不能先分页再排序，也不能只统计第一批1000。

目录可能包含授权项目内guest看不到的issueID；它们仅在服务器快照内作为数据，最终issue过滤仍统一HiddenFrom/OwnOnly对应SQL，绝不能把原目录返回/记日志/用作总数。counts本身不授予可见性。读错/重复ID/负数/非升序目录作为内部协议错误整体失败，不能默默覆盖。

±排序沿现有UUID同方向tie-breaker、精确int64 cursor规则，0是有效值非null；无新group_by attachment。项目列表保留archived/sub_issue，工作区内成员列表保留自身的关系及项目过滤能力；空页重新读total仍消费同一目录，不增加工作区通用列表路由或成员子分组。设置保存不能接受旧版本客户端未定义的假排序码；TS比较器/验证类型与当前API生成同步，P7再挂菜单。

## 4. 资源与容量

不新增常驻缓存、冗余计数列或修复任务；所有值由同事务asset事实推导，不需要补偿一致性。代价：仅附件排序时要传完整非零目录。asset计数SQL的输入projectIDs/issueIDs每批至多1000；Issue列表SQL接收的是完整内存目录，不受此资产查询批次上限截断。跨批次的全局issue排序按UUID合并，不能按项目遍历导致after游标跳过。实现将已排序去重的项目每1000个分批；对同一全局after游标，每批取前1000个issue计数，按issue UUID归并后返回全局前1000。每批前N行必然覆盖全集合前N行，下一次仅推进全局issue游标，所有批次使用调用方同一快照。项目数≤1000时每页只有一条SQL；跨批目录不得重复issue ID，读错整体失败。1001项目交错的2002行三页及1001工作项真实SQL分别验证全集无漏。

保持现有普通API的15秒期限及容量测试客户端5秒预算；冷统计1000/1001和10000个有附件工作项的列表≤5秒，带多个项目/guest/分组/两方向，记录SQL数、输入规模与真实耗时。正向夹具通过API（容量数据允许复用既有有标注的内部批量容量fixture，但不得冒称API数据）；若现有合同无法满足规模，提交证据上报，不擅自加持久投影或跨模块JOIN。

## 5. 反例与回归

计数包括别人的活附件但排除描述、未完成、未绑定、评论、草稿、deleted；同名不同ID、ID顺序与数量顺序相反、零值与相同计数的tie、每方向跨三页、空页total、group/subgroup同issue多cell去重、不同成员relation及guest隐藏，均要精确断言。

并发新增/删除置于快照开始前后，证明一次读内count/排序/cursor一致；跨请求变化按既有link_count限制记录，不误称固定快照。定向变异删除uploaded/bound/deleted过滤、截断1000、page后排序、跳过OwnOnly、float64 cursor、吞端口错误，各有真正失败。
