package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

type user struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"passwordHash"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"createdAt"`
}

type session struct {
	Token     string    `json:"token"`
	UserID    string    `json:"userId"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type publishJob struct {
	ID        string          `json:"id"`
	UserID    string          `json:"userId"`
	Content   sourceContent   `json:"content"`
	Status    string          `json:"status"`
	Summary   string          `json:"summary"`
	Results   []publishResult `json:"results"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

type platformAccount struct {
	ID           string    `json:"id"`
	UserID       string    `json:"userId"`
	PlatformID   string    `json:"platformId"`
	PlatformName string    `json:"platformName"`
	AccountName  string    `json:"accountName"`
	Status       string    `json:"status"`
	AccessToken  string    `json:"accessToken,omitempty"`
	RefreshToken string    `json:"refreshToken,omitempty"`
	ExpiresAt    time.Time `json:"expiresAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type publicPlatformAccount struct {
	ID           string    `json:"id"`
	PlatformID   string    `json:"platformId"`
	PlatformName string    `json:"platformName"`
	AccountName  string    `json:"accountName"`
	Status       string    `json:"status"`
	ExpiresAt    time.Time `json:"expiresAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type storeData struct {
	Users            []user            `json:"users"`
	Sessions         []session         `json:"sessions"`
	Jobs             []publishJob      `json:"jobs"`
	PlatformAccounts []platformAccount `json:"platformAccounts"`
}

type store struct {
	mu   sync.Mutex
	path string
	data storeData
}

type appServer struct {
	store *store
	queue chan string
}

type adaptResponse struct {
	Items []adaptedContent `json:"items"`
}

type authRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authResponse struct {
	User  publicUser `json:"user"`
	Token string     `json:"token"`
}

type platformAccountRequest struct {
	PlatformID   string `json:"platformId"`
	AccountName  string `json:"accountName"`
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

type publicUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

type publishResponse struct {
	Job     publishJob      `json:"job"`
	Summary string          `json:"summary"`
	Items   []publishResult `json:"items"`
}

type publishJobDetail struct {
	publishJob
	AdaptedContents []adaptedContent `json:"adaptedContents"`
}

type contextKey string

const (
	sessionCookieName = "mcp_session"
	sessionTTL        = 24 * time.Hour
	currentUserKey    = contextKey("currentUser")
)

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
	dataStore, err := newStore(filepath.Join("data", "store.json"))
	if err != nil {
		log.Fatal(err)
	}

	server := &appServer{
		store: dataStore,
		queue: make(chan string, 100),
	}
	go server.runPublishWorker()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth/register", server.handleRegister)
	mux.HandleFunc("/api/auth/login", server.handleLogin)
	mux.HandleFunc("/api/auth/logout", server.withAuth(server.handleLogout))
	mux.HandleFunc("/api/me", server.withAuth(server.handleMe))
	mux.HandleFunc("/api/adapt", server.withAuth(server.handleAdapt))
	mux.HandleFunc("/api/publish/mock", server.withAuth(server.handleMockPublish))
	mux.HandleFunc("/api/publish/jobs", server.withAuth(server.handleJobs))
	mux.HandleFunc("/api/platform-accounts", server.withAuth(server.handlePlatformAccounts))
	mux.Handle("/", http.FileServer(http.Dir(".")))

	log.Println("server started at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", withCORS(mux)))
}

func newStore(path string) (*store, error) {
	s := &store{path: path, data: storeData{}}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return s, s.saveLocked()
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(content) == 0 {
		return s, nil
	}
	if err := json.Unmarshal(content, &s.data); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *store) saveLocked() error {
	content, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, content, 0o600)
}

func (s *store) createUser(username string, password string) (user, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	username = strings.TrimSpace(username)
	if username == "" || strings.TrimSpace(password) == "" {
		return user{}, errors.New("username and password are required")
	}
	for _, existingUser := range s.data.Users {
		if strings.EqualFold(existingUser.Username, username) {
			return user{}, errors.New("username already exists")
		}
	}

	role := "creator"
	if len(s.data.Users) == 0 {
		role = "admin"
	}

	newUser := user{
		ID:           newID("user"),
		Username:     username,
		PasswordHash: hashPassword(password),
		Role:         role,
		CreatedAt:    time.Now(),
	}
	s.data.Users = append(s.data.Users, newUser)
	return newUser, s.saveLocked()
}

func (s *store) authenticate(username string, password string) (user, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, existingUser := range s.data.Users {
		if strings.EqualFold(existingUser.Username, strings.TrimSpace(username)) && existingUser.PasswordHash == hashPassword(password) {
			return existingUser, nil
		}
	}
	return user{}, errors.New("invalid username or password")
}

func (s *store) createSession(userID string) (session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	newSession := session{
		Token:     newID("session"),
		UserID:    userID,
		ExpiresAt: time.Now().Add(sessionTTL),
	}
	s.data.Sessions = append(s.data.Sessions, newSession)
	return newSession, s.saveLocked()
}

func (s *store) deleteSession(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessions := s.data.Sessions[:0]
	for _, item := range s.data.Sessions {
		if item.Token != token {
			sessions = append(sessions, item)
		}
	}
	s.data.Sessions = sessions
	return s.saveLocked()
}

func (s *store) userBySession(token string) (user, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	var sessionUserID string
	for _, item := range s.data.Sessions {
		if item.Token == token && item.ExpiresAt.After(now) {
			sessionUserID = item.UserID
			break
		}
	}
	if sessionUserID == "" {
		return user{}, false
	}

	for _, item := range s.data.Users {
		if item.ID == sessionUserID {
			return item, true
		}
	}
	return user{}, false
}

func (s *store) createJob(userID string, content sourceContent) (publishJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	job := publishJob{
		ID:        newID("job"),
		UserID:    userID,
		Content:   content,
		Status:    "queued",
		Summary:   "发布任务已进入队列",
		Results:   pendingResults(),
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.data.Jobs = append(s.data.Jobs, job)
	return job, s.saveLocked()
}

func (s *store) updateJob(jobID string, update func(*publishJob)) (publishJob, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for index := range s.data.Jobs {
		if s.data.Jobs[index].ID == jobID {
			update(&s.data.Jobs[index])
			s.data.Jobs[index].UpdatedAt = time.Now()
			return s.data.Jobs[index], true, s.saveLocked()
		}
	}
	return publishJob{}, false, nil
}

func (s *store) jobsByUser(userID string, role string) []publishJob {
	s.mu.Lock()
	defer s.mu.Unlock()

	jobs := make([]publishJob, 0, len(s.data.Jobs))
	for _, job := range s.data.Jobs {
		if role == "admin" || job.UserID == userID {
			jobs = append(jobs, job)
		}
	}
	return jobs
}

func (s *store) jobByID(userID string, role string, jobID string) (publishJob, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, job := range s.data.Jobs {
		if job.ID == jobID && (role == "admin" || job.UserID == userID) {
			return job, true
		}
	}
	return publishJob{}, false
}

func (s *store) platformAccountsByUser(userID string) []publicPlatformAccount {
	s.mu.Lock()
	defer s.mu.Unlock()

	accounts := make([]publicPlatformAccount, 0, len(adapters))
	for _, adapter := range adapters {
		account := publicPlatformAccount{
			PlatformID:   adapter.ID,
			PlatformName: adapter.Name,
			Status:       "disconnected",
		}

		for _, existing := range s.data.PlatformAccounts {
			if existing.UserID == userID && existing.PlatformID == adapter.ID {
				account = toPublicPlatformAccount(existing)
				break
			}
		}

		accounts = append(accounts, account)
	}

	return accounts
}

func (s *store) upsertPlatformAccount(userID string, request platformAccountRequest) (publicPlatformAccount, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	adapter, ok := platformAdapterByID(request.PlatformID)
	if !ok {
		return publicPlatformAccount{}, errors.New("unsupported platform")
	}
	accountName := strings.TrimSpace(request.AccountName)
	if accountName == "" {
		return publicPlatformAccount{}, errors.New("account name is required")
	}

	now := time.Now()
	for index := range s.data.PlatformAccounts {
		if s.data.PlatformAccounts[index].UserID == userID && s.data.PlatformAccounts[index].PlatformID == adapter.ID {
			s.data.PlatformAccounts[index].PlatformName = adapter.Name
			s.data.PlatformAccounts[index].AccountName = accountName
			s.data.PlatformAccounts[index].Status = "connected"
			s.data.PlatformAccounts[index].AccessToken = request.AccessToken
			s.data.PlatformAccounts[index].RefreshToken = request.RefreshToken
			s.data.PlatformAccounts[index].ExpiresAt = now.Add(30 * 24 * time.Hour)
			s.data.PlatformAccounts[index].UpdatedAt = now
			return toPublicPlatformAccount(s.data.PlatformAccounts[index]), s.saveLocked()
		}
	}

	account := platformAccount{
		ID:           newID("platform"),
		UserID:       userID,
		PlatformID:   adapter.ID,
		PlatformName: adapter.Name,
		AccountName:  accountName,
		Status:       "connected",
		AccessToken:  request.AccessToken,
		RefreshToken: request.RefreshToken,
		ExpiresAt:    now.Add(30 * 24 * time.Hour),
		UpdatedAt:    now,
	}
	s.data.PlatformAccounts = append(s.data.PlatformAccounts, account)
	return toPublicPlatformAccount(account), s.saveLocked()
}

func (s *store) deletePlatformAccount(userID string, platformID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	accounts := s.data.PlatformAccounts[:0]
	for _, account := range s.data.PlatformAccounts {
		if !(account.UserID == userID && account.PlatformID == platformID) {
			accounts = append(accounts, account)
		}
	}
	s.data.PlatformAccounts = accounts
	return s.saveLocked()
}

func (s *appServer) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var request authRequest
	if !readJSON(w, r, &request) {
		return
	}

	createdUser, err := s.store.createUser(request.Username, request.Password)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	createdSession, err := s.store.createSession(createdUser.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create session failed")
		return
	}

	setSessionCookie(w, createdSession)
	writeJSON(w, http.StatusCreated, authResponse{User: toPublicUser(createdUser), Token: createdSession.Token})
}

func (s *appServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var request authRequest
	if !readJSON(w, r, &request) {
		return
	}

	currentUser, err := s.store.authenticate(request.Username, request.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	createdSession, err := s.store.createSession(currentUser.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create session failed")
		return
	}

	setSessionCookie(w, createdSession)
	writeJSON(w, http.StatusOK, authResponse{User: toPublicUser(currentUser), Token: createdSession.Token})
}

func (s *appServer) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		_ = s.store.deleteSession(cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}

func (s *appServer) handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]publicUser{"user": toPublicUser(mustCurrentUser(r))})
}

func (s *appServer) handleAdapt(w http.ResponseWriter, r *http.Request) {
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

func (s *appServer) handleMockPublish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	content, ok := readSourceContent(w, r)
	if !ok {
		return
	}

	currentUser := mustCurrentUser(r)
	job, err := s.store.createJob(currentUser.ID, content)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create publish job failed")
		return
	}
	s.queue <- job.ID

	writeJSON(w, http.StatusAccepted, publishResponse{Job: job, Summary: job.Summary, Items: job.Results})
}

func (s *appServer) handleJobs(w http.ResponseWriter, r *http.Request) {
	currentUser := mustCurrentUser(r)
	switch r.Method {
	case http.MethodGet:
		jobID := r.URL.Query().Get("id")
		if jobID != "" {
			job, ok := s.store.jobByID(currentUser.ID, currentUser.Role, jobID)
			if !ok {
				writeError(w, http.StatusNotFound, "job not found")
				return
			}
			writeJSON(w, http.StatusOK, map[string]publishJobDetail{"job": buildJobDetail(job)})
			return
		}

		writeJSON(w, http.StatusOK, map[string][]publishJob{"jobs": s.store.jobsByUser(currentUser.ID, currentUser.Role)})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *appServer) handlePlatformAccounts(w http.ResponseWriter, r *http.Request) {
	currentUser := mustCurrentUser(r)

	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string][]publicPlatformAccount{"accounts": s.store.platformAccountsByUser(currentUser.ID)})
	case http.MethodPost:
		var request platformAccountRequest
		if !readJSON(w, r, &request) {
			return
		}

		account, err := s.store.upsertPlatformAccount(currentUser.ID, request)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]publicPlatformAccount{"account": account})
	case http.MethodDelete:
		platformID := r.URL.Query().Get("platformId")
		if _, ok := platformAdapterByID(platformID); !ok {
			writeError(w, http.StatusBadRequest, "unsupported platform")
			return
		}

		if err := s.store.deletePlatformAccount(currentUser.ID, platformID); err != nil {
			writeError(w, http.StatusInternalServerError, "disconnect platform failed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"message": "platform disconnected"})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func buildJobDetail(job publishJob) publishJobDetail {
	return publishJobDetail{
		publishJob:      job,
		AdaptedContents: adaptContent(job.Content),
	}
}

func (s *appServer) runPublishWorker() {
	for jobID := range s.queue {
		_, ok, err := s.store.updateJob(jobID, func(job *publishJob) {
			job.Status = "running"
			job.Summary = "正在模拟发布到各平台"
			job.Results = pendingResults()
		})
		if err != nil || !ok {
			log.Println("start publish job failed:", jobID, err)
			continue
		}

		time.Sleep(700 * time.Millisecond)

		_, _, err = s.store.updateJob(jobID, func(job *publishJob) {
			job.Status = "done"
			job.Summary, job.Results = buildPublishResults(job.Content)
		})
		if err != nil {
			log.Println("finish publish job failed:", jobID, err)
		}
	}
}

func (s *appServer) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil || strings.TrimSpace(cookie.Value) == "" {
			writeError(w, http.StatusUnauthorized, "login required")
			return
		}

		currentUser, ok := s.store.userBySession(cookie.Value)
		if !ok {
			writeError(w, http.StatusUnauthorized, "login required")
			return
		}

		ctx := context.WithValue(r.Context(), currentUserKey, currentUser)
		next(w, r.WithContext(ctx))
	}
}

func readSourceContent(w http.ResponseWriter, r *http.Request) (sourceContent, bool) {
	var content sourceContent
	if !readJSON(w, r, &content) {
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

func readJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return false
	}
	return true
}

func adaptContent(content sourceContent) []adaptedContent {
	items := make([]adaptedContent, 0, len(adapters))
	for _, adapter := range adapters {
		items = append(items, adapter.Adapt(content))
	}
	return items
}

func buildPublishResults(content sourceContent) (string, []publishResult) {
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

	if !hasBody {
		return "内容不完整，已生成草稿", results
	}
	return "全部平台模拟发布完成", results
}

func pendingResults() []publishResult {
	results := make([]publishResult, 0, len(adapters))
	for _, adapter := range adapters {
		results = append(results, publishResult{
			PlatformName: adapter.Name,
			Status:       "pending",
			Message:      "等待平台响应",
			Detail:       "模拟任务已创建",
		})
	}
	return results
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

func platformAdapterByID(platformID string) (platformAdapter, bool) {
	for _, adapter := range adapters {
		if adapter.ID == platformID {
			return adapter, true
		}
	}
	return platformAdapter{}, false
}

func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

func toPublicPlatformAccount(account platformAccount) publicPlatformAccount {
	return publicPlatformAccount{
		ID:           account.ID,
		PlatformID:   account.PlatformID,
		PlatformName: account.PlatformName,
		AccountName:  account.AccountName,
		Status:       account.Status,
		ExpiresAt:    account.ExpiresAt,
		UpdatedAt:    account.UpdatedAt,
	}
}

func newID(prefix string) string {
	buffer := make([]byte, 12)
	if _, err := rand.Read(buffer); err != nil {
		return prefix + "-" + time.Now().Format("20060102150405")
	}
	return prefix + "-" + hex.EncodeToString(buffer)
}

func toPublicUser(item user) publicUser {
	return publicUser{
		ID:       item.ID,
		Username: item.Username,
		Role:     item.Role,
	}
}

func mustCurrentUser(r *http.Request) user {
	currentUser, _ := r.Context().Value(currentUserKey).(user)
	return currentUser
}

func setSessionCookie(w http.ResponseWriter, item session) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    item.Token,
		Path:     "/",
		Expires:  item.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
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
