const titleInput = document.querySelector("#titleInput");
const bodyInput = document.querySelector("#bodyInput");
const tagsInput = document.querySelector("#tagsInput");
const previewTitle = document.querySelector("#previewTitle");
const previewBody = document.querySelector("#previewBody");
const previewTags = document.querySelector("#previewTags");
const previewTips = document.querySelector("#platformTips");
const platformTabs = document.querySelector("#platformTabs");
const wordCount = document.querySelector("#wordCount");
const accountStatus = document.querySelector("#accountStatus");
const authPanel = document.querySelector("#authPanel");
const authMessage = document.querySelector("#authMessage");
const usernameInput = document.querySelector("#usernameInput");
const passwordInput = document.querySelector("#passwordInput");
const loginButton = document.querySelector("#loginButton");
const registerButton = document.querySelector("#registerButton");
const logoutButton = document.querySelector("#logoutButton");
const publishButton = document.querySelector("#publishButton");
const publishResults = document.querySelector("#publishResults");
const publishSummary = document.querySelector("#publishSummary");
const refreshHistoryButton = document.querySelector("#refreshHistoryButton");
const historyList = document.querySelector("#historyList");
const recordDetailPanel = document.querySelector("#recordDetailPanel");
const recordDetail = document.querySelector("#recordDetail");
const closeDetailButton = document.querySelector("#closeDetailButton");
let activePlatformId = PLATFORM_ADAPTERS[0].id;
let latestAdaptedContents = [];
let latestSourceContent = null;
let currentUser = null;
let currentJobId = null;

function parseTags(value) {
  return value
    .split(/[,，]/)
    .map((tag) => tag.trim())
    .filter(Boolean);
}

async function requestAdaptedContents(content) {
  try {
    const response = await fetch("/api/adapt", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(content),
    });

    if (!response.ok) {
      throw new Error("adapt api failed");
    }

    const data = await response.json();
    return data.items;
  } catch (error) {
    if (!currentUser) {
      authMessage.textContent = "登录后将使用 Go 后端适配接口，当前使用前端兜底预览。";
    }
    return adaptContentForPlatforms(content);
  }
}

async function renderPreview() {
  const content = createSourceContent({
    title: titleInput.value,
    body: bodyInput.value,
    tags: parseTags(tagsInput.value),
  });
  const adaptedContents = await requestAdaptedContents(content);
  const activeContent = adaptedContents.find((item) => item.platformId === activePlatformId) || adaptedContents[0];
  latestAdaptedContents = adaptedContents;
  latestSourceContent = content;

  renderPlatformTabs(adaptedContents);
  previewTitle.textContent = activeContent.title;
  previewBody.textContent = activeContent.body;
  wordCount.textContent = `${content.body.length} 字`;

  previewTags.replaceChildren(
    ...activeContent.tags.map((tag) => {
      const tagElement = document.createElement("span");
      tagElement.className = "tag";
      tagElement.textContent = `#${tag}`;
      return tagElement;
    }),
  );

  previewTips.replaceChildren(
    ...activeContent.tips.map((tip) => {
      const tipElement = document.createElement("p");
      tipElement.textContent = tip;
      return tipElement;
    }),
  );
}

function renderPublishResults(results) {
  publishResults.replaceChildren(
    ...results.map((result) => {
      const resultCard = document.createElement("article");
      resultCard.className = `publish-result ${result.status}`;

      const title = document.createElement("h3");
      title.textContent = result.platformName;

      const message = document.createElement("p");
      message.textContent = result.message;

      const meta = document.createElement("span");
      meta.textContent = result.detail;

      resultCard.append(title, message, meta);
      return resultCard;
    }),
  );
}

function renderAuthState() {
  if (currentUser) {
    accountStatus.textContent = `${currentUser.username} · ${currentUser.role}`;
    authPanel.classList.add("hidden");
    logoutButton.hidden = false;
    publishButton.disabled = false;
    authMessage.textContent = "";
  } else {
    accountStatus.textContent = "未登录";
    authPanel.classList.remove("hidden");
    logoutButton.hidden = true;
    publishButton.disabled = true;
  }
}

async function requestJSON(url, options = {}) {
  const response = await fetch(url, {
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...(options.headers || {}),
    },
    ...options,
  });
  const data = await response.json().catch(() => ({}));

  if (!response.ok) {
    throw new Error(data.error || "request failed");
  }

  return data;
}

async function loadCurrentUser() {
  try {
    const data = await requestJSON("/api/me", {
      method: "GET",
    });
    currentUser = data.user;
  } catch (error) {
    currentUser = null;
  }
  renderAuthState();
}

async function submitAuth(mode) {
  authMessage.textContent = "";
  const username = usernameInput.value.trim();
  const password = passwordInput.value.trim();

  if (!username || !password) {
    authMessage.textContent = "请输入用户名和密码。";
    return;
  }

  try {
    const data = await requestJSON(`/api/auth/${mode}`, {
      method: "POST",
      body: JSON.stringify({ username, password }),
    });
    currentUser = data.user;
    passwordInput.value = "";
    renderAuthState();
    await renderPreview();
    await loadHistory();
  } catch (error) {
    authMessage.textContent = error.message;
  }
}

async function logout() {
  try {
    await requestJSON("/api/auth/logout", {
      method: "POST",
      body: "{}",
    });
  } catch (error) {
    // Local state still clears if the session has already expired.
  }

  currentUser = null;
  currentJobId = null;
  renderAuthState();
  renderHistory([]);
}

async function requestMockPublish(content) {
  try {
    const response = await fetch("/api/publish/mock", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(content),
    });

    if (!response.ok) {
      throw new Error("publish api failed");
    }

    const data = await response.json();
    currentJobId = data.job.id;
    return data;
  } catch (error) {
    const hasBody = bodyInput.value.trim().length > 0;

    return {
      summary: hasBody ? "全部平台模拟发布完成" : "内容不完整，已生成草稿",
      items: latestAdaptedContents.map((item, index) => ({
        platformName: item.platformName,
        status: hasBody ? "success" : "warning",
        message: hasBody ? "模拟发布成功" : "缺少正文，已保存为草稿",
        detail: `任务编号 MOCK-${index + 1}-${item.platformId.toUpperCase()}`,
      })),
    };
  }
}

async function simulatePublish() {
  if (!currentUser) {
    authMessage.textContent = "请先注册或登录，再创建发布任务。";
    return;
  }
  if (!latestAdaptedContents.length) {
    await renderPreview();
  }

  publishButton.disabled = true;
  publishButton.textContent = "发布中...";
  publishSummary.textContent = "正在模拟发布到各平台";

  const pendingResults = latestAdaptedContents.map((item) => ({
    platformName: item.platformName,
    status: "pending",
    message: "等待平台响应",
    detail: "模拟任务已创建",
  }));
  renderPublishResults(pendingResults);

  window.setTimeout(async () => {
    const publishResponse = await requestMockPublish(latestSourceContent);

    publishButton.disabled = false;
    publishButton.textContent = "一键模拟发布";
    publishSummary.textContent = publishResponse.summary;
    renderPublishResults(publishResponse.items);
    pollJobUntilDone();
    loadHistory();
  }, 700);
}

async function pollJobUntilDone() {
  if (!currentJobId) {
    return;
  }

  try {
    const data = await requestJSON(`/api/publish/jobs?id=${encodeURIComponent(currentJobId)}`, {
      method: "GET",
    });
    publishSummary.textContent = data.job.summary;
    renderPublishResults(data.job.results);
    renderRecordDetail(data.job);
    await loadHistory();

    if (data.job.status === "queued" || data.job.status === "running") {
      window.setTimeout(pollJobUntilDone, 500);
    }
  } catch (error) {
    authMessage.textContent = error.message;
  }
}

async function loadHistory() {
  if (!currentUser) {
    renderHistory([]);
    return;
  }

  try {
    const data = await requestJSON("/api/publish/jobs", {
      method: "GET",
    });
    renderHistory(data.jobs || []);
  } catch (error) {
    authMessage.textContent = error.message;
  }
}

function renderHistory(jobs) {
  if (!jobs.length) {
    historyList.replaceChildren(createEmptyHistory());
    return;
  }

  historyList.replaceChildren(
    ...jobs
      .slice()
      .reverse()
      .map((job) => {
        const item = document.createElement("article");
        item.className = "history-item";

        const title = document.createElement("h3");
        title.textContent = job.content.title;

        const meta = document.createElement("p");
        meta.textContent = `${job.status} · ${job.summary}`;

        const time = document.createElement("span");
        time.textContent = new Date(job.updatedAt).toLocaleString();

        const action = document.createElement("button");
        action.className = "secondary-button";
        action.type = "button";
        action.textContent = "查看详情";
        action.addEventListener("click", () => loadRecordDetail(job.id));

        item.append(title, meta, time, action);
        return item;
      }),
  );
}

function createEmptyHistory() {
  const empty = document.createElement("p");
  empty.className = "empty-history";
  empty.textContent = "暂无发布记录。";
  return empty;
}

async function loadRecordDetail(jobId) {
  try {
    const data = await requestJSON(`/api/publish/jobs?id=${encodeURIComponent(jobId)}`, {
      method: "GET",
    });
    renderRecordDetail(data.job);
  } catch (error) {
    authMessage.textContent = error.message;
  }
}

function renderRecordDetail(job) {
  if (!job) {
    return;
  }

  recordDetailPanel.classList.remove("hidden");
  recordDetail.replaceChildren(
    createDetailSection("原始内容", [
      ["标题", job.content.title],
      ["正文", job.content.body || "未填写正文"],
      ["标签", (job.content.tags || []).join(", ") || "未填写标签"],
    ]),
    createAdaptedDetail(job.adaptedContents || []),
    createPublishResultDetail(job.results || []),
  );
}

function createDetailSection(title, rows) {
  const section = document.createElement("section");
  section.className = "detail-section";

  const heading = document.createElement("h3");
  heading.textContent = title;
  section.append(heading);

  rows.forEach(([label, value]) => {
    const row = document.createElement("div");
    row.className = "detail-row";

    const labelElement = document.createElement("strong");
    labelElement.textContent = label;

    const valueElement = document.createElement("p");
    valueElement.textContent = value;

    row.append(labelElement, valueElement);
    section.append(row);
  });

  return section;
}

function createAdaptedDetail(adaptedContents) {
  const section = document.createElement("section");
  section.className = "detail-section";

  const heading = document.createElement("h3");
  heading.textContent = "平台适配内容";
  section.append(heading);

  adaptedContents.forEach((content) => {
    const card = document.createElement("article");
    card.className = "adapted-detail-card";

    const title = document.createElement("h4");
    title.textContent = `${content.platformName} · ${content.title}`;

    const body = document.createElement("p");
    body.textContent = content.body;

    const tags = document.createElement("span");
    tags.textContent = (content.tags || []).map((tag) => `#${tag}`).join(" ");

    card.append(title, body, tags);
    section.append(card);
  });

  return section;
}

function createPublishResultDetail(results) {
  const section = document.createElement("section");
  section.className = "detail-section";

  const heading = document.createElement("h3");
  heading.textContent = "发布结果";
  section.append(heading);

  results.forEach((result) => {
    const row = document.createElement("div");
    row.className = `result-detail-row ${result.status}`;
    row.textContent = `${result.platformName}：${result.message} · ${result.detail}`;
    section.append(row);
  });

  return section;
}

function renderPlatformTabs(adaptedContents) {
  platformTabs.replaceChildren(
    ...adaptedContents.map((item) => {
      const button = document.createElement("button");
      button.type = "button";
      button.className = item.platformId === activePlatformId ? "platform-tab active" : "platform-tab";
      button.textContent = item.platformName;
      button.addEventListener("click", () => {
        activePlatformId = item.platformId;
        renderPreview();
      });
      return button;
    }),
  );
}

[titleInput, bodyInput, tagsInput].forEach((input) => {
  input.addEventListener("input", renderPreview);
});

loginButton.addEventListener("click", () => submitAuth("login"));
registerButton.addEventListener("click", () => submitAuth("register"));
logoutButton.addEventListener("click", logout);
publishButton.addEventListener("click", simulatePublish);
refreshHistoryButton.addEventListener("click", loadHistory);
closeDetailButton.addEventListener("click", () => {
  recordDetailPanel.classList.add("hidden");
});

loadCurrentUser().then(loadHistory);
renderPreview();
renderPublishResults(
  PLATFORM_ADAPTERS.map((platform) => ({
    platformName: platform.name,
    status: "idle",
    message: "等待发布",
    detail: platform.tone,
  })),
);
