# TDX HTTP API Reference

本文档基于 `extend/httpserver` 当前代码生成，覆盖全部 79 个 GET 路由的请求参数与
响应 `data` 类型。

## 基本信息

- 默认地址：`http://localhost:18080`
- 请求方法：全部为 `GET`
- 参数位置：URL Query
- 请求体：无
- 业务响应 Content-Type：`application/json`；`/doc` 返回 `text/html; charset=utf-8`；框架生成的 404/405 响应可能为 `text/plain`
- 字符编码：UTF-8
- 鉴权：服务本身不提供鉴权

扩展行情接口 `/ex/*` 仅在配置 `TDX_EXHQ_HOSTS` 后注册，否则返回 HTTP 404。

## 统一响应

除 `/doc` 文档页外，所有已注册接口使用以下响应外壳：

```json
{
  "code": 0,
  "msg": "ok",
  "data": {}
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `code` | integer | `0` 成功，`1` 失败 |
| `msg` | string | 成功时为 `ok`，失败时为错误信息 |
| `data` | any | 成功数据；失败时为 `null` |

### HTTP 状态码

| HTTP 状态 | `code` | 场景 |
| --- | --- | --- |
| `200` | `0` | 请求成功 |
| `200` | - | `/doc` 返回独立 HTML 文档页 |
| `200` | `1` | 上游行情服务返回错误；当前实现保持 HTTP 200 |
| `400` | `1` | 缺少参数、数字格式错误、交易所不支持 |
| `404` | - | 路由不存在，或扩展行情未启用 |
| `405` | - | 请求方法不是 GET |
| `503` | `1` | `/ready` 检测到标准或扩展行情连接未就绪 |

错误示例：

```json
{
  "code": 1,
  "msg": "参数 code 不能为空",
  "data": null
}
```

## 数据约定

- 除路由表明确写“无”或使用方括号标记的可选参数外，所有 Query 参数均为必填参数，
  没有分页默认值。
- 标准行情模型多数没有 JSON Tag，字段名保持 Go 导出字段格式，例如 `Count`、
  `List`、`Code`。
- 股票详情聚合模型使用 lowerCamelCase，例如 `code`、`blocks`、`tdxCode`。
- 扩展行情模型使用 lowerCamelCase，例如 `market`、`shortName`、`preClose`。
- `protocol.Price` 在 JSON 中是整数，单位为厘；换算为元需要除以 `1000`。
- `time.Time` 在 JSON 中编码为 RFC3339 字符串。
- `[]byte` 在 JSON 中编码为 Base64 字符串。
- `Exchange` 在响应中是整数：`0` 深圳、`1` 上海、`2` 北京。

## 公共请求参数

| 参数 | Query 类型 | 取值/格式 | 说明 |
| --- | --- | --- | --- |
| `exchange` | string | `sh`、`sz`、`bj`，大小写均可 | 标准市场；仅部分接口单独传递 |
| `code` | string | `600519`、`sh600519`、`sz000001` | 证券代码；有 `exchange` 参数时通常传 6 位代码 |
| `codes` | string | `sz000001,sh600519` | 多个代码使用英文逗号分隔 |
| `type` | uint8 | `0`～`11` | K 线类型，见下表 |
| `start` | uint16/uint32 | 非负整数 | 分页起始位置；`/ex/instruments` 使用 uint32，其余分页接口使用 uint16 |
| `count` | uint16 | `0`～`65535` | 本次获取数量 |
| `date` | string/uint32 | `YYYYMMDD` | 标准历史接口按字符串读取，扩展接口按 uint32 读取 |
| `date2` | uint32 | `YYYYMMDD` | 扩展历史 K 线结束日期 |
| `since` | string | `YYYYMMDD` | 可选、包含式 K 线起始日期；仅支持标明该参数的全量接口 |
| `adjust` | string | `none`、`qfq`、`hfq` | 可选日线复权类型，默认 `none`，大小写不敏感 |
| `market` | uint8 | 来自 `/ex/markets` | 扩展行情市场编号 |
| `category` | uint8 | 按接口定义 | 扩展行情类别或 K 线类型 |
| `file` | string | 文件名 | 板块或报表文件名 |
| `filename` | string | F10 文件名 | 从 `/company/category` 的 `filename` 获取 |
| `length` | uint32 | 非负整数 | F10 内容读取长度 |

### K 线类型

| `type`/`category` | 周期 |
| --- | --- |
| `0` | 5 分钟 |
| `1` | 15 分钟 |
| `2` | 30 分钟 |
| `3` | 60 分钟 |
| `4` | 日 K 变体 |
| `5` | 周 K |
| `6` | 月 K |
| `7` | 1 分钟 |
| `8` | 1 分钟变体 |
| `9` | 日 K |
| `10` | 季 K |
| `11` | 年 K |

## 接口列表

表中“响应 data”只描述统一响应外壳中的 `data` 字段；`/doc` 是不使用该外壳的
独立 HTML 页面。

### 存活、就绪与文档

| 路径 | 请求参数 | 响应 data | 说明 |
| --- | --- | --- | --- |
| `GET /` | 无 | `{status:string}` | HTTP 进程存活检查，不检查上游 |
| `GET /ready` | 无 | `ReadyStatus` | 检查标准行情及已启用的扩展行情连接 |
| `GET /doc` | 无 | HTML 页面 | 离线 API 请求参数与响应文档，不使用 JSON 外壳 |

`ReadyStatus` 成功示例：

```json
{
  "status": "ready",
  "standard": true,
  "extendedEnabled": false
}
```

### 证券代码与数量

| 路径 | 请求参数 | 响应 data | 说明 |
| --- | --- | --- | --- |
| `GET /count` | `exchange:string` | `CountResp` | 指定交易所证券数量 |
| `GET /code` | `exchange:string`, `start:uint16` | `CodeResp` | 分页获取证券代码 |
| `GET /code/all` | `exchange:string` | `CodeResp` | 获取指定交易所全部代码 |
| `GET /code/stocks` | 无 | `string[]` | 全部股票代码 |
| `GET /code/stocks/detail` | 无 | `StockDetail[]` | 全部 A 股代码、名称、板块和行业 |
| `GET /code/etfs` | 无 | `string[]` | 全部 ETF 代码 |
| `GET /code/indexes` | 无 | `string[]` | 全部指数代码 |

### 实时行情、竞价与财务

| 路径 | 请求参数 | 响应 data | 说明 |
| --- | --- | --- | --- |
| `GET /quote` | `codes:string` | `Quote[]` | 批量实时行情 |
| `GET /call_auction` | `code:string` | `CallAuctionResp` | 集合竞价数据 |
| `GET /gbbq` | `code:string` | `GbbqResp` | 除权除息及股本变更 |
| `GET /finance` | `exchange:string`, `code:string` | `FinanceInfo` | 财务信息 |
| `GET /company/category` | `exchange:string`, `code:string` | `CompanyCategory[]` | F10 文件目录 |
| `GET /company/content` | `exchange:string`, `code:string`, `filename:string`, `start:uint32`, `length:uint32` | `string` | F10 文件内容 |

### 分时与成交

| 路径 | 请求参数 | 响应 data | 说明 |
| --- | --- | --- | --- |
| `GET /minute` | `code:string` | `MinuteResp` | 当日分时 |
| `GET /minute/history` | `date:string`, `code:string` | `MinuteResp` | 历史分时 |
| `GET /trade` | `code:string`, `start:uint16`, `count:uint16` | `TradeResp` | 当日分笔成交分页 |
| `GET /trade/all` | `code:string` | `TradeResp` | 当日全部分笔成交 |
| `GET /trade/history` | `date:string`, `code:string`, `start:uint16`, `count:uint16` | `TradeResp` | 历史分笔成交分页 |
| `GET /trade/history/day` | `date:string`, `code:string` | `TradeResp` | 指定日期全部成交 |

### 股票 K 线

以下接口的响应 `data` 均为 `KlineResp`。

| 路径 | 请求参数 | 周期 |
| --- | --- | --- |
| `GET /kline` | `type:uint8`, `code:string`, `start:uint16`, `count:uint16` | 由 `type` 指定，分页 |
| `GET /kline/all` | `type:uint8`, `code:string`, [`since:string`] | 由 `type` 指定，全部或按日期截断 |
| `GET /kline/minute` | `code:string`, `start:uint16`, `count:uint16` | 1 分钟，分页 |
| `GET /kline/minute/all` | `code:string` | 1 分钟，全部 |
| `GET /kline/minute/241` | `code:string`, [`since:string`] | 包含 09:30 集合竞价分钟的 1 分钟 K 线 |
| `GET /kline/5minute` | `code:string`, `start:uint16`, `count:uint16` | 5 分钟，分页 |
| `GET /kline/5minute/all` | `code:string` | 5 分钟，全部 |
| `GET /kline/15minute` | `code:string`, `start:uint16`, `count:uint16` | 15 分钟，分页 |
| `GET /kline/15minute/all` | `code:string` | 15 分钟，全部 |
| `GET /kline/30minute` | `code:string`, `start:uint16`, `count:uint16` | 30 分钟，分页 |
| `GET /kline/30minute/all` | `code:string` | 30 分钟，全部 |
| `GET /kline/60minute` | `code:string`, `start:uint16`, `count:uint16` | 60 分钟，分页 |
| `GET /kline/60minute/all` | `code:string` | 60 分钟，全部 |
| `GET /kline/day` | `code:string`, `start:uint16`, `count:uint16` | 日 K，分页 |
| `GET /kline/day/all` | `code:string`, [`since:string`], [`adjust:string`] | 日 K，支持日期截断及前/后复权 |
| `GET /kline/week` | `code:string`, `start:uint16`, `count:uint16` | 周 K，分页 |
| `GET /kline/week/all` | `code:string` | 周 K，全部 |
| `GET /kline/month` | `code:string`, `start:uint16`, `count:uint16` | 月 K，分页 |
| `GET /kline/month/all` | `code:string` | 月 K，全部 |
| `GET /kline/quarter` | `code:string`, `start:uint16`, `count:uint16` | 季 K，分页 |
| `GET /kline/quarter/all` | `code:string` | 季 K，全部 |
| `GET /kline/year` | `code:string`, `start:uint16`, `count:uint16` | 年 K，分页 |
| `GET /kline/year/all` | `code:string` | 年 K，全部 |

`since` 是包含式下界，例如 `since=20240101` 仅返回 2024-01-01 及之后的数据。
`adjust=qfq` 返回前复权日线，`adjust=hfq` 返回后复权日线。为保持复权锚点稳定，
复权请求即使设置 `since` 也会先计算完整历史，再过滤响应。

### 指数 K 线

以下接口的响应 `data` 均为 `KlineResp`。

| 路径 | 请求参数 | 周期 |
| --- | --- | --- |
| `GET /index` | `type:uint8`, `code:string`, `start:uint16`, `count:uint16` | 由 `type` 指定，分页 |
| `GET /index/all` | `type:uint8`, `code:string`, [`since:string`] | 由 `type` 指定，全部或按日期截断 |
| `GET /index/minute` | `code:string`, `start:uint16`, `count:uint16` | 1 分钟，分页 |
| `GET /index/5minute` | `code:string`, `start:uint16`, `count:uint16` | 5 分钟，分页 |
| `GET /index/15minute` | `code:string`, `start:uint16`, `count:uint16` | 15 分钟，分页 |
| `GET /index/30minute` | `code:string`, `start:uint16`, `count:uint16` | 30 分钟，分页 |
| `GET /index/60minute` | `code:string`, `start:uint16`, `count:uint16` | 60 分钟，分页 |
| `GET /index/day` | `code:string`, `start:uint16`, `count:uint16` | 日 K，分页 |
| `GET /index/day/all` | `code:string` | 日 K，全部 |
| `GET /index/week/all` | `code:string` | 周 K，全部 |
| `GET /index/month/all` | `code:string` | 月 K，全部 |
| `GET /index/quarter/all` | `code:string` | 季 K，全部 |
| `GET /index/year/all` | `code:string` | 年 K，全部 |

### 板块、报表与配置

| 路径 | 请求参数 | 响应 data | 说明 |
| --- | --- | --- | --- |
| `GET /block/data` | `file:string` | `Block[]` | 解析板块成分 |
| `GET /block/data/index` | `file:string` | `Block[]` | 板块成分及指数代码 |
| `GET /block/file` | `file:string` | Base64 string | 原始板块文件字节 |
| `GET /report/file` | `file:string` | Base64 string | 原始报表文件字节 |
| `GET /zhb/files` | 无 | `object<string, Base64 string>` | `zhb.zip` 解压后的文件映射 |
| `GET /tdx/zs` | 无 | `TdxZs[]` | 通达信指数配置 |
| `GET /tdx/bk` | 无 | `TdxBk[]` | 板块简称与全称映射 |
| `GET /tdx/stat` | 无 | `TdxStat[]` | 综合统计指标 |
| `GET /tdx/stat2` | 无 | `TdxStat2[]` | 资金流向及板块归属 |
| `GET /tdx/xgsg` | 无 | `TdxXgsg[]` | 新股申购信息 |
| `GET /tdx/hy` | 无 | `TdxHy[]` | 通达信/申万行业归属 |
| `GET /spblock` | 无 | `SpBlock[]` | 大型指数等特殊板块成分 |

常用 `file`：`block_zs.dat`、`block_gn.dat`、`block_fg.dat`、`block_hy.dat`、
`block.dat`。常用报表文件为 `zhb.zip`。

### 扩展行情

需要先配置 `TDX_EXHQ_HOSTS`。除 `/ex/count` 外，扩展行情模型使用 lowerCamelCase
JSON 字段。

| 路径 | 请求参数 | 响应 data | 说明 |
| --- | --- | --- | --- |
| `GET /ex/markets` | 无 | `ExMarket[]` | 市场列表 |
| `GET /ex/count` | 无 | integer | 全部扩展品种数量 |
| `GET /ex/instruments` | `start:uint32`, `count:uint16` | `ExInstrument[]` | 品种列表分页 |
| `GET /ex/quote` | `market:uint8`, `code:string` | `ExQuote` | 单品种五档行情 |
| `GET /ex/quote_list` | `market:uint8`, `category:uint8`, `start:uint16`, `count:uint16` | `ExQuoteListItem[]` | 批量行情；常见 `category=2` 港股、`3` 期货 |
| `GET /ex/bars` | `category:uint8`, `market:uint8`, `code:string`, `start:uint16`, `count:uint16` | `ExKline[]` | K 线分页；`category` 使用 K 线类型 |
| `GET /ex/minute` | `market:uint8`, `code:string` | `ExMinuteTick[]` | 当日分时 |
| `GET /ex/minute/hist` | `market:uint8`, `code:string`, `date:uint32` | `ExMinuteTick[]` | 历史分时 |
| `GET /ex/trade` | `market:uint8`, `code:string`, `start:uint16`, `count:uint16` | `ExTradeTick[]` | 当日分笔成交 |
| `GET /ex/trade/hist` | `market:uint8`, `code:string`, `date:uint32`, `start:uint16`, `count:uint16` | `ExTradeTick[]` | 历史分笔成交 |
| `GET /ex/bars/range` | `market:uint8`, `code:string`, `date:uint32`, `date2:uint32` | `ExRangeKline[]` | 日期区间 K 线 |

## 响应数据模型

### CountResp 与 CodeResp

| 模型 | 字段 |
| --- | --- |
| `CountResp` | `Count:integer` |
| `CodeResp` | `Count:integer`, `List:Code[]` |
| `Code` | `Name:string`, `Code:string`, `Multiple:integer`, `Decimal:integer`, `LastPrice:number` |

### StockDetail

`GET /code/stocks/detail` 返回上海、深圳、北京全部 A 股，按 `code` 升序排列。结果
使用 30 分钟服务端快照缓存，首次请求需要批量下载市场、板块和行业配置。

| 模型 | 字段 |
| --- | --- |
| `StockDetail` | `code:string`, `name:string`, `blocks:StockBlock[]`, `industry:StockIndustry` |
| `StockBlock` | `name:string`, `index:string`, `category:string` |
| `StockIndustry` | `tdxCode:string`, `swCode:string` |

`StockBlock.category` 可取 `concept`、`industry`、`style`、`index`、`general`、
`special`。`StockBlock.index` 无可用板块指数映射时为空；行业字段返回通达信新行业和
申万行业代码，不解析为名称。

### Quote

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `Exchange` | integer | `0` 深、`1` 沪、`2` 北 |
| `Code` | string | 6 位证券代码 |
| `Active1` | integer | 活跃度 |
| `Kline` | `Kline` | 当前行情对应 K 线数据 |
| `ServerTime` | string | 服务器时间 |
| `Intuition` | integer | 现量 |
| `InsideDish` | integer | 内盘 |
| `OuterDisc` | integer | 外盘 |
| `BuyLevel` | `PriceLevel[5]` | 买一至买五 |
| `SellLevel` | `PriceLevel[5]` | 卖一至卖五 |
| `Rate` | number | 涨速 |
| `Active2` | integer | 活跃度 |
| `ReversedBytes0`～`ReversedBytes9` | integer | 协议保留字段 |

`PriceLevel`：`Buy:boolean`、`Price:integer`（厘）、`Number:integer`。

### CallAuctionResp

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `Count` | integer | 数据数量 |
| `List` | `CallAuction[]` | 集合竞价列表 |

`CallAuction`：`Time:string`、`Price:integer`（厘）、`Match:integer`、
`Unmatched:integer`、`Flag:integer`。`Flag=1` 表示未匹配量为买单，`-1` 表示卖单。

### GbbqResp

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `Count` | integer | 数据数量 |
| `List` | `Gbbq[]` | 股本变更列表 |

`Gbbq`：`Code:string`、`Time:string`、`Category:integer`、`C1:number`、
`C2:number`、`C3:number`、`C4:number`。`Category` 常见值为
`2`、`3`、`5`、`7`、`8`、`9`、`10`，各 `C*` 含义随类别变化。

### FinanceInfo

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `Market` | integer | 市场编号 |
| `Code` | string | 证券代码 |
| `LiuTongGuBen` | number | 流通股本 |
| `Province` | integer | 地域码 |
| `Industry` | integer | 行业码 |
| `UpdatedDate` | integer | 更新日期 `YYYYMMDD` |
| `IPODate` | integer | 上市日期 `YYYYMMDD` |
| `ZongGuBen` | number | 总股本 |
| `GuoJiaGu` | number | 国家股 |
| `FaQiRenFaRenGu` | number | 发起人法人股 |
| `FaRenGu` | number | 法人股 |
| `BGu` | number | B 股 |
| `HGu` | number | H 股 |
| `ZhiGongGu` | number | 职工股 |
| `ZongZiChan` | number | 总资产 |
| `LiuDongZiChan` | number | 流动资产 |
| `GuDingZiChan` | number | 固定资产 |
| `WuXingZiChan` | number | 无形资产 |
| `GuDongRenShu` | number | 股东人数 |
| `LiuDongFuZhai` | number | 流动负债 |
| `ChangQiFuZhai` | number | 长期负债 |
| `ZiBenGongJiJin` | number | 资本公积金 |
| `JingZiChan` | number | 净资产 |
| `ZhuYingShouRu` | number | 主营收入 |
| `ZhuYingLiRun` | number | 主营利润 |
| `YingShouZhangKuan` | number | 应收账款 |
| `YingYeLiRun` | number | 营业利润 |
| `TouZiShouYi` | number | 投资收益 |
| `JingYingXianJinLiu` | number | 经营现金流 |
| `ZongXianJinLiu` | number | 总现金流 |
| `CunHuo` | number | 存货 |
| `LiRunZongHe` | number | 利润总额 |
| `ShuiHouLiRun` | number | 税后利润 |
| `JingLiRun` | number | 净利润 |
| `WeiFenLiRun` | number | 未分配利润 |
| `BaoLiu1`、`BaoLiu2` | number | 协议保留字段 |

### CompanyCategory

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `name` | string | 分类名称 |
| `filename` | string | F10 文件名 |
| `start` | integer | 文件内偏移 |
| `length` | integer | 内容长度 |

### MinuteResp

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `Count` | integer | 数据数量 |
| `List` | `PriceNumber[]` | 分时数据 |

`PriceNumber`：`Time:string`（`HH:mm`）、`Price:integer`（厘）、
`Number:integer`（成交量，手）。

### TradeResp

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `Count` | integer | 数据数量 |
| `List` | `Trade[]` | 成交列表 |

`Trade`：`Time:string`、`Price:integer`（厘）、`Volume:integer`（手）、
`Status:integer`、`Number:integer`。`Status=0` 买，`1` 卖，`2` 中性/汇总。

### KlineResp

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `Count` | integer | K 线数量 |
| `List` | `Kline[]` | K 线列表 |

| Kline 字段 | 类型 | 说明 |
| --- | --- | --- |
| `Last` | integer | 昨收，单位厘 |
| `Open` | integer | 开盘，单位厘 |
| `High` | integer | 最高，单位厘 |
| `Low` | integer | 最低，单位厘 |
| `Close` | integer | 收盘/最新，单位厘 |
| `Order` | integer | 成交单数，部分接口可能无值 |
| `Volume` | integer | 成交量，手 |
| `Amount` | integer | 成交额，协议 `Price` 类型 |
| `Time` | string | RFC3339 时间 |
| `UpCount` | integer | 上涨数量，指数有效 |
| `DownCount` | integer | 下跌数量，指数有效 |

### 板块与配置模型

| 模型 | 字段 |
| --- | --- |
| `Block` | `Name:string`, `Index:string`, `Type:integer`, `Codes:string[]` |
| `TdxZs` | `Name:string`, `Code:string`, `Type:integer`, `SubType:integer`, `Ref:string` |
| `TdxBk` | `Short:string`, `Full:string` |
| `TdxStat` | `Market:integer`, `Code:string`, `Date:string`, `PETTM:number`, `TrendDays:integer`, `ChangePct:number`, `PEStatic:number`, `DivYield:number`, `Chg5:number`, `Chg10:number`, `Chg20:number`, `Chg60:number`, `ChgYTD:number`, `Fields:string[]` |
| `TdxStat2` | `Market:integer`, `Code:string`, `Date:string`, `BlockIndex:string`, `Amount:number`, `AmountPrev:number`, `IPOPrice:number`, `High52W:number`, `Low52W:number`, `Fields:string[]` |
| `TdxXgsg` | `Market:integer`, `Code:string`, `Date:string`, `IssuePrice:number`, `Name:string`, `Fields:string[]` |
| `TdxHy` | `Market:integer`, `Code:string`, `TdxHy:string`, `SwHy:string` |
| `SpBlock` | `Name:string`, `Codes:string[]` |

### 扩展行情模型

| 模型 | 字段 |
| --- | --- |
| `ExMarket` | `market:integer`, `category:integer`, `name:string`, `shortName:string` |
| `ExInstrument` | `category:integer`, `market:integer`, `code:string`, `name:string`, `desc:string` |
| `ExMinuteTick` | `hour:integer`, `minute:integer`, `price:number`, `avgPrice:number`, `volume:integer`, `openInterest:integer` |
| `ExTradeTick` | `hour:integer`, `minute:integer`, `second:integer`, `price:integer`, `volume:integer`, `zengCang:integer`, `nature:integer`, `natureName:string`, `direction:integer` |
| `ExKline` | `datetime:string`, `open:number`, `high:number`, `low:number`, `close:number`, `position:integer`, `trade:integer`, `price:number`, `amount:number` |
| `ExRangeKline` | `datetime:string`, `open:number`, `high:number`, `low:number`, `close:number`, `position:integer`, `trade:integer`, `settlementPrice:number` |

`ExTradeTick.direction`：`1` 买、`-1` 卖、`0` 中性。

`ExQuote` 字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `market` | integer | 市场编号 |
| `code` | string | 品种代码 |
| `preClose` | number | 昨收/昨结 |
| `open`、`high`、`low`、`price` | number | 开高低现价 |
| `kaiCang` | integer | 开仓量 |
| `zongLiang` | integer | 总量 |
| `xianLiang` | integer | 现量 |
| `neiPan`、`waiPan` | integer | 内盘、外盘 |
| `chiCang` | integer | 持仓量 |
| `bid`、`ask` | `number[5]` | 五档买价、卖价 |
| `bidVol`、`askVol` | `integer[5]` | 五档买量、卖量 |

`ExQuoteListItem` 与 `ExQuote` 的盘口数组相同，其他字段为：`market`、`code`、
`preClose`、`open`、`high`、`low`、`price`、`zongLiang`、`amount`、`inner`、
`outer`、`chiCang`。

## 请求示例

### 存活、就绪与文档

```bash
curl "http://localhost:18080/"
curl -i "http://localhost:18080/ready"
curl "http://localhost:18080/doc"
```

推荐直接在浏览器打开 `http://localhost:18080/doc` 阅读完整文档。

### 代码与实时行情

```bash
curl "http://localhost:18080/count?exchange=sh"
curl "http://localhost:18080/code?exchange=sz&start=0"
curl "http://localhost:18080/code/stocks/detail"
curl "http://localhost:18080/quote?codes=sz000001,sh600519"
```

### 分时、成交与 K 线

```bash
curl "http://localhost:18080/minute?code=sz000001"
curl "http://localhost:18080/trade?code=sz000001&start=0&count=100"
curl "http://localhost:18080/kline/day?code=sh600519&start=0&count=100"
curl "http://localhost:18080/kline/all?type=9&code=sh600519&since=20240101"
curl "http://localhost:18080/kline/day/all?code=sh600519&adjust=qfq&since=20240101"
curl "http://localhost:18080/kline/minute/241?code=sz000001&since=20260801"
curl "http://localhost:18080/index/all?type=9&code=sh000001&since=20240101"
```

### 财务与 F10

```bash
curl "http://localhost:18080/finance?exchange=sh&code=600519"
curl "http://localhost:18080/company/category?exchange=sh&code=600519"
curl "http://localhost:18080/company/content?exchange=sh&code=600519&filename=600519.txt&start=0&length=1000"
```

### 板块与报表

```bash
curl "http://localhost:18080/block/data?file=block_gn.dat"
curl "http://localhost:18080/block/data/index?file=block_zs.dat"
curl "http://localhost:18080/report/file?file=zhb.zip"
```

### 扩展行情

```bash
curl "http://localhost:18080/ex/markets"
curl "http://localhost:18080/ex/instruments?start=0&count=100"
curl "http://localhost:18080/ex/quote?market=47&code=600519"
curl "http://localhost:18080/ex/bars?category=9&market=47&code=600519&start=0&count=100"
```

扩展市场编号和代码应以当前服务器返回的 `/ex/markets`、`/ex/instruments` 为准。
