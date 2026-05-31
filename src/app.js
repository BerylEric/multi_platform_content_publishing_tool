const titleInput = document.querySelector("#titleInput");
const bodyInput = document.querySelector("#bodyInput");
const tagsInput = document.querySelector("#tagsInput");
const previewTitle = document.querySelector("#previewTitle");
const previewBody = document.querySelector("#previewBody");
const previewTags = document.querySelector("#previewTags");
const previewTips = document.querySelector("#platformTips");
const platformTabs = document.querySelector("#platformTabs");
const wordCount = document.querySelector("#wordCount");
let activePlatformId = PLATFORM_ADAPTERS[0].id;

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

renderPreview();
