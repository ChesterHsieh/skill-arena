# 🛠️ 如何從零打造一個 Agent Skill：以 Apache Flink 技能為例

> **SOP 版本：** v1.0 | **適用對象：** AI 工程師、技術主管、平台開發者
> **目標：** 標準化 Agent Skill 的建立流程，讓每一個 Skill 都具備高品質、可測試、易維護的特性

---

## 📌 什麼是 Agent Skill？

Agent Skill 是一份給 AI Agent 閱讀的「任務說明書」，告訴它在特定情境下該如何完成工作。
它的核心是一個 `SKILL.md` 檔案，結構清晰、可擴充，並支援測試與迭代優化。

**一個 Skill 解決的核心問題：**
_「如何讓 AI Agent 在面對特定領域任務時，像一位專家一樣可靠地執行？」_

---

## 🗺️ 整體 SOP 流程

```
需求定義  →  目錄設計  →  SKILL.md 撰寫  →  測試案例設計  →  執行 & 評估  →  迭代優化  →  發布
   ①            ②             ③                  ④               ⑤              ⑥          ⑦
```

---

## ① 需求定義：搞清楚 Skill 要做什麼

在動手寫任何程式碼之前，先回答這四個問題：

| 問題 | Flink Skill 的答案範例 |
|------|------------------------|
| **這個 Skill 要讓 AI 做什麼？** | 協助撰寫 Apache Flink 的 DataStream / Table API 程式碼，解讀 Flink Job 架構，以及診斷常見 checkpoint 問題 |
| **什麼時候應該觸發這個 Skill？** | 使用者提到 Flink、stream processing、Kafka source、watermark、backpressure、state backend 等關鍵字時 |
| **預期輸出格式是什麼？** | 可直接執行的 Java/Python Flink 程式碼 + 架構解釋 + 注意事項 |
| **需要建立測試案例嗎？** | 是（程式碼生成類 Skill 有明確的驗證標準，適合建立測試） |

> 💡 **原則：** 需求越具體，Skill 越精準。「幫我寫 Flink 程式碼」遠不如「幫我用 Flink DataStream API 處理 Kafka 的 JSON 事件流，並實作 tumbling window 聚合」。

---

## ② 目錄設計：建立正確的檔案結構

```
flink-skill/
├── SKILL.md                    # 核心：指令 + 觸發條件（必填）
├── references/
│   ├── datastream-api.md       # DataStream API 參考文件
│   ├── table-api.md            # Table API 參考文件
│   └── troubleshooting.md      # 常見錯誤排查指南
├── scripts/
│   └── validate_flink_job.py   # 驗證 Flink Job 設定的腳本
├── assets/
│   └── flink-job-template.java # 標準 Job 模板
└── evals/
    └── evals.json              # 測試案例定義
```

**設計原則（Progressive Disclosure）：**

```
Layer 1：SKILL.md 的 metadata（name + description）
  → 約 100 字，永遠在 Agent 的 context 中
  → 決定 Agent 是否呼叫這個 Skill

Layer 2：SKILL.md 的 body（完整指令）
  → 500 行以內，Skill 觸發後才載入
  → 包含執行邏輯與分支判斷

Layer 3：references/ 與 scripts/（延伸資源）
  → 按需載入，無大小限制
  → 腳本可直接執行而無需完整載入
```

---

## ③ SKILL.md 撰寫：Skill 的靈魂

### 基本格式

```markdown
---
name: flink-skill
description: >
  Apache Flink 串流處理開發助理。協助撰寫 DataStream API、
  Table API 程式碼，設計 Flink Job 架構，診斷 checkpoint、
  backpressure、watermark 等問題。
  當使用者提到 Flink、stream processing、Kafka source、
  CEP、state backend、exactly-once 等詞彙時，務必使用此 Skill。
---

# Flink Skill

你是一位 Apache Flink 專家，擅長設計高吞吐、低延遲的串流處理系統...
```

### Description 的撰寫黃金法則

**❌ 太模糊（容易漏觸發）**
```
description: 幫助使用者處理串流資料
```

**✅ 精確 + 積極（提高觸發準確率）**
```
description: >
  Apache Flink 串流處理開發助理。包含 DataStream API、
  Table API、CEP、State Management、Checkpoint 調優。
  當使用者提到 Flink、streaming、Kafka pipeline、
  watermark、backpressure 時，務必使用此 Skill。
```

> 💡 **關鍵洞察：** `description` 是 Agent 決定是否呼叫 Skill 的**唯一依據**。
> 寫得太保守 → Agent 漏掉使用時機
> 寫得太廣泛 → Agent 在不相關場景誤觸發

### SKILL.md Body 的結構建議

```markdown
## 工作流程

依照使用者的需求，選擇以下路徑：

**A. 程式碼生成**
1. 確認 Flink 版本（1.14 / 1.17 / 1.19）
2. 確認 Source 類型（Kafka / File / Custom）
3. 確認處理邏輯（filter / map / window / join）
4. 生成程式碼並加上必要的注釋

**B. 架構診斷**
1. 請使用者提供 Job Graph 或錯誤日誌
2. 讀取 references/troubleshooting.md
3. 給出診斷結論與建議

## 輸出格式

ALWAYS 使用以下格式回應：

### 程式碼
[可執行的 Java/Python 程式碼，附完整 import]

### 說明
[架構決策說明，200 字以內]

### 注意事項
[潛在的效能問題、相容性問題]
```

---

## ④ 測試案例設計：讓 Skill 可被驗證

### evals.json 格式

```json
{
  "skill_name": "flink-skill",
  "evals": [
    {
      "id": 1,
      "prompt": "我需要一個 Flink job，從 Kafka topic 'user-events' 讀取 JSON，過濾出 event_type='purchase' 的事件，並用 5 分鐘 tumbling window 計算每個 user_id 的總消費金額，結果寫回 Kafka topic 'purchase-summary'",
      "expected_output": "完整可執行的 DataStream API Java 程式碼，包含 KafkaSource、filter、keyBy、window、aggregate、KafkaSink"
    },
    {
      "id": 2,
      "prompt": "我的 Flink job 一直出現 checkpoint timeout，log 顯示 'Checkpoint 123 expired before completing'，請幫我診斷",
      "expected_output": "列出常見原因（GC pause、慢 operator、state 過大）並提供具體排查步驟"
    },
    {
      "id": 3,
      "prompt": "如何在 Flink 實作 watermark 來處理亂序事件？事件最多可能延遲 30 秒",
      "expected_output": "解釋 event time vs processing time 概念，提供 BoundedOutOfOrdernessWatermarks 的程式碼範例"
    }
  ]
}
```

### 測試案例設計原則

| 類型 | 目的 | Flink 範例 |
|------|------|------------|
| **核心功能** | 驗證最常見使用情境 | 生成基本的 Kafka → Flink → Kafka pipeline |
| **邊界情境** | 測試複雜或不常見的需求 | 實作 late data 處理 + side output |
| **錯誤診斷** | 驗證問題排查能力 | 從 exception stack trace 找出根因 |

---

## ⑤ 執行與評估：看 Skill 實際表現

### 執行方式

對每一個測試案例，同時執行兩個版本進行對比：

```
測試案例 #1
    ├── 有 Skill 的版本  →  outputs/with_skill/
    └── 無 Skill 的版本  →  outputs/without_skill/
```

### 量化評估指標（Assertions）

```json
{
  "assertions": [
    {
      "text": "程式碼包含完整的 KafkaSource 設定（bootstrap.servers, topic, group.id）",
      "type": "contains"
    },
    {
      "text": "程式碼可通過 Java 語法檢查（無明顯語法錯誤）",
      "type": "syntax_valid"
    },
    {
      "text": "包含 window 函數（TumblingEventTimeWindows 或同類）",
      "type": "contains"
    },
    {
      "text": "有加入錯誤處理或注意事項說明",
      "type": "quality"
    }
  ]
}
```

### 評估維度

```
定性評估（人工）          量化評估（自動）
─────────────────         ────────────────
□ 程式碼是否可直接執行？   □ Assertion 通過率
□ 架構說明是否清晰？       □ 執行時間
□ 是否符合 Flink 最佳實踐？□ Token 用量
□ 有 Skill vs 無 Skill 的差異明顯嗎？
```

---

## ⑥ 迭代優化：讓 Skill 越來越好

### 迭代循環

```
執行測試  →  收集反饋  →  更新 SKILL.md  →  再次執行測試  →  ...
```

### 常見問題與改善策略

| 問題症狀 | 可能原因 | 改善方式 |
|----------|----------|----------|
| 每次測試都重複生成相同的 helper 腳本 | 缺少 bundled script | 將重複邏輯移至 `scripts/` 目錄 |
| 輸出格式不一致 | SKILL.md 的輸出格式定義不夠明確 | 在 SKILL.md 加入固定模板 |
| Agent 常常沒有觸發 Skill | Description 關鍵字不夠 | 補充同義詞與使用情境描述 |
| 遇到進階 API 時輸出品質下降 | References 文件不足 | 在 `references/` 加入更詳細的文件 |

### 優化 Description 的工具化方法

建立 20 個觸發測試案例（10 個應觸發 + 10 個不應觸發），用自動化工具跑 A/B 測試：

```bash
# 觸發測試範例
✅ 應觸發：「我的 Flink job 在處理 10 萬 QPS 時出現 backpressure，怎麼調優？」
✅ 應觸發：「幫我設計一個 exactly-once 的 Flink pipeline」
❌ 不應觸發：「Kafka consumer 的 offset reset 怎麼設定？」
❌ 不應觸發：「Spark Streaming 和 Flink 哪個好？」
```

---

## ⑦ 發布：打包與部署

### 打包 Skill

```bash
python -m scripts.package_skill ./flink-skill
# 輸出：flink-skill.skill（ZIP 格式，可直接安裝）
```

### 最終 Checklist

```
□ SKILL.md 的 name 和 description 已填寫
□ SKILL.md body 不超過 500 行（或有清楚的分層結構）
□ 所有 references 檔案都有在 SKILL.md 中指引使用時機
□ evals.json 有至少 3 個測試案例
□ 有 Skill vs 無 Skill 的輸出對比結果符合預期
□ Description 的觸發準確率測試通過
□ .skill 打包檔案可成功安裝
```

---

## 🔑 核心設計哲學

> **「告訴 AI 為什麼，而不只是告訴它怎麼做。」**

1. **說明原因** — 解釋每個步驟背後的邏輯，讓 Agent 能舉一反三
2. **漸進式揭露** — 把常用知識放在 SKILL.md，罕用知識放在 references/
3. **可測試性** — 每個 Skill 都應該可以被量化評估
4. **不過度約束** — 避免用過多的 "MUST/ALWAYS"，改用說理的方式引導行為

---

## 📊 一張圖看懂整個流程

```
┌─────────────────────────────────────────────────────┐
│                  Agent Skill 生命週期                 │
├──────────┬──────────┬──────────┬──────────┬──────────┤
│  需求定義 │ 目錄設計 │ 撰寫 SKILL│  測試驗證 │  持續優化 │
│          │          │          │          │          │
│ 4 個問題 │ 3 層架構 │ 描述 +   │ evals +  │  迭代 +  │
│ 確認範疇 │ 漸進揭露 │ 流程 +   │ 量化指標 │ description│
│          │          │ 輸出格式  │          │  優化     │
└──────────┴──────────┴──────────┴──────────┴──────────┘
```

---

*🔖 如果這份 SOP 對你有幫助，歡迎按讚分享！*
*有任何 Agent Skill 開發的問題，歡迎在留言區討論 👇*
