const titleInput = document.querySelector("#titleInput");
const bodyInput = document.querySelector("#bodyInput");
const tagsInput = document.querySelector("#tagsInput");
const previewTitle = document.querySelector("#previewTitle");
const previewBody = document.querySelector("#previewBody");
const previewTags = document.querySelector("#previewTags");
const previewTips = document.querySelector("#platformTips");
const platformTabs = document.querySelector("#platformTabs");
const wordCount = document.querySelector("#wordCount");
const publishButton = document.querySelector("#publishButton");
const publishResults = document.querySelector("#publishResults");
const publishSummary = document.querySelector("#publishSummary");
let activePlatformId = PLATFORM_ADAPTERS[0].id;
let latestAdaptedContents = [];

function parseTags(value) {
  return value
    .split(/[,，]/)
    .map((tag) => tag.trim())
    .filter(Boolean);
}

function renderPreview() {
  const content = createSourceContent({
    title: titleInput.value,
    body: bodyInput.value,
    tags: parseTags(tagsInput.value),
  });
  const adaptedContents = adaptContentForPlatforms(content);
  const activeContent = adaptedContents.find((item) => item.platformId === activePlatformId) || adaptedContents[0];
  latestAdaptedContents = adaptedContents;

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

function simulatePublish() {
  if (!latestAdaptedContents.length) {
    renderPreview();
  }

  const hasBody = bodyInput.value.trim().length > 0;
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

  window.setTimeout(() => {
    const results = latestAdaptedContents.map((item, index) => ({
      platformName: item.platformName,
      status: hasBody ? "success" : "warning",
      message: hasBody ? "模拟发布成功" : "缺少正文，已保存为草稿",
      detail: `任务编号 MOCK-${index + 1}-${item.platformId.toUpperCase()}`,
    }));

    publishButton.disabled = false;
    publishButton.textContent = "一键模拟发布";
    publishSummary.textContent = hasBody ? "全部平台模拟发布完成" : "内容不完整，已生成草稿";
    renderPublishResults(results);
  }, 700);
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

publishButton.addEventListener("click", simulatePublish);

renderPreview();
renderPublishResults(
  PLATFORM_ADAPTERS.map((platform) => ({
    platformName: platform.name,
    status: "idle",
    message: "等待发布",
    detail: platform.tone,
  })),
);
