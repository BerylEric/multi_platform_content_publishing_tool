package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

type sourceContent struct {
	Title string   `json:"title"`
	Body  string   `json:"body"`
	Tags  []string `json:"tags"`
}

type adaptedContent struct {
	PlatformID   string   `json:"platformId"`
	PlatformName string   `json:"platformName"`
	Title        string   `json:"title"`
	Body         string   `json:"body"`
	Tags         []string `json:"tags"`
	Tips         []string `json:"tips"`
}

type publishResult struct {
	PlatformName string `json:"platformName"`
	Status       string `json:"status"`
	Message      string `json:"message"`
	Detail       string `json:"detail"`
}

type adaptResponse struct {
	Items []adaptedContent `json:"items"`
}

type publishResponse struct {
	Summary string          `json:"summary"`
	Items   []publishResult `json:"items"`
}

type platformAdapter struct {
	ID    string
	Name  string
	Tone  string
	Adapt func(sourceContent) adaptedContent
}

var adapters = []platformAdapter{
	{
		ID:   "wechat",
		Name: "公众号",
		Tone: "结构清晰、适合长文阅读",
		Adapt: func(content sourceContent) adaptedContent {
			return adaptedContent{
				PlatformID:   "wechat",
				PlatformName: "公众号",
				Title:        content.Title,
				Body:         strings.Join([]string{"导语：" + content.Title, "", fallbackBody(content.Body, "这里是正文内容，可继续补充完整观点、案例和结论。"), "", "如果这篇内容对你有帮助，欢迎收藏并持续关注后续更新。"}, "\n"),
				Tags:         limitTags(content.Tags, 3),
				Tips:         []string{"首段突出价值点", "正文保留分段", "文末适合放关注引导"},
			}
		},
	},
	{
		ID:   "zhihu",
		Name: "知乎",
		Tone: "问题导向、强调观点和论证",
		Adapt: func(content sourceContent) adaptedContent {
			return adaptedContent{
				PlatformID:   "zhihu",
				PlatformName: "知乎",
				Title:        content.Title + "：我的方法和思考",
				Body:         strings.Join([]string{"先说结论：" + content.Title + "的关键在于把内容拆成可复用的结构。", "", fallbackBody(content.Body, "这里可以补充背景、分析过程和个人经验。"), "", "你也可以根据自己的平台和受众，继续调整表达方式。"}, "\n"),
				Tags:         limitTags(content.Tags, 5),
				Tips:         []string{"标题适合带问题意识", "正文突出经验和理由", "结尾可补充讨论问题"},
			}
		},
	},
	{
		ID:   "bilibili",
		Name: "B站",
		Tone: "轻松直接、适合动态或专栏引流",
		Adapt: func(content sourceContent) adaptedContent {
			return adaptedContent{
				PlatformID:   "bilibili",
				PlatformName: "B站",
				Title:        "【创作分享】" + content.Title,
				Body:         strings.Join([]string{content.Title, "", fallbackBody(content.Body, "这是一条适合 B 站动态或专栏的内容简介。"), "", "欢迎在评论区交流你的做法。"}, "\n"),
				Tags:         limitTags(content.Tags, 6),
				Tips:         []string{"标题可以更有记忆点", "正文适合加入互动引导", "标签可覆盖内容领域和受众"},
			}
		},
	},
	{
		ID:   "xiaohongshu",
		Name: "小红书",
		Tone: "种草笔记、强调场景和关键词",
		Adapt: func(content sourceContent) adaptedContent {
			bodyParts := []string{content.Title, "", fallbackBody(content.Body, "今天分享一个适合创作者提升发布效率的小方法。")}
			if tagText := buildTagText(content.Tags); tagText != "" {
				bodyParts = append(bodyParts, "", tagText)
			}

			return adaptedContent{
				PlatformID:   "xiaohongshu",
				PlatformName: "小红书",
				Title:        content.Title + "｜实用分享",
				Body:         strings.Join(bodyParts, "\n"),
				Tags:         limitTags(content.Tags, 8),
				Tips:         []string{"开头给出使用场景", "正文适合短句和要点", "标签建议覆盖搜索关键词"},
			}
		},
	},
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/adapt", handleAdapt)
	mux.HandleFunc("/api/publish/mock", handleMockPublish)
	mux.Handle("/", http.FileServer(http.Dir(".")))

	log.Println("server started at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", withCORS(mux)))
}

func handleAdapt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	content, ok := readSourceContent(w, r)
	if !ok {
		return
	}

	writeJSON(w, http.StatusOK, adaptResponse{Items: adaptContent(content)})
}

func handleMockPublish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	content, ok := readSourceContent(w, r)
	if !ok {
		return
	}

	items := adaptContent(content)
	results := make([]publishResult, 0, len(items))
	hasBody := strings.TrimSpace(content.Body) != ""

	for index, item := range items {
		result := publishResult{
			PlatformName: item.PlatformName,
			Status:       "success",
			Message:      "模拟发布成功",
			Detail:       buildTaskID(index, item.PlatformID),
		}
		if !hasBody {
			result.Status = "warning"
			result.Message = "缺少正文，已保存为草稿"
		}
		results = append(results, result)
	}

	summary := "全部平台模拟发布完成"
	if !hasBody {
		summary = "内容不完整，已生成草稿"
	}

	writeJSON(w, http.StatusOK, publishResponse{Summary: summary, Items: results})
}

func readSourceContent(w http.ResponseWriter, r *http.Request) (sourceContent, bool) {
	var content sourceContent
	if err := json.NewDecoder(r.Body).Decode(&content); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return sourceContent{}, false
	}

	content.Title = strings.TrimSpace(content.Title)
	content.Body = strings.TrimSpace(content.Body)
	content.Tags = normalizeTags(content.Tags)
	if content.Title == "" {
		content.Title = "未命名内容"
	}

	return content, true
}

func adaptContent(content sourceContent) []adaptedContent {
	items := make([]adaptedContent, 0, len(adapters))
	for _, adapter := range adapters {
		items = append(items, adapter.Adapt(content))
	}
	return items
}

func normalizeTags(tags []string) []string {
	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(tags))

	for _, tag := range tags {
		cleanTag := strings.TrimSpace(tag)
		if cleanTag == "" {
			continue
		}
		if _, exists := seen[cleanTag]; exists {
			continue
		}
		seen[cleanTag] = struct{}{}
		normalized = append(normalized, cleanTag)
	}

	return normalized
}

func limitTags(tags []string, max int) []string {
	if len(tags) <= max {
		return tags
	}
	return tags[:max]
}

func fallbackBody(body string, fallback string) string {
	if strings.TrimSpace(body) == "" {
		return fallback
	}
	return body
}

func buildTagText(tags []string) string {
	if len(tags) == 0 {
		return ""
	}

	parts := make([]string, 0, len(tags))
	for _, tag := range tags {
		parts = append(parts, "#"+tag)
	}
	return strings.Join(parts, " ")
}

func buildTaskID(index int, platformID string) string {
	return "任务编号 MOCK-" + time.Now().Format("20060102") + "-" + strings.ToUpper(platformID) + "-" + string(rune('A'+index))
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Println("write response failed:", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
