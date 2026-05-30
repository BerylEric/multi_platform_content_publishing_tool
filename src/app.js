const titleInput = document.querySelector("#titleInput");
const bodyInput = document.querySelector("#bodyInput");
const tagsInput = document.querySelector("#tagsInput");
const previewTitle = document.querySelector("#previewTitle");
const previewBody = document.querySelector("#previewBody");
const previewTags = document.querySelector("#previewTags");
const wordCount = document.querySelector("#wordCount");

function parseTags(value) {
  return value
    .split(/[,，]/)
    .map((tag) => tag.trim())
    .filter(Boolean);
}

function renderPreview() {
  const title = titleInput.value.trim();
  const body = bodyInput.value.trim();
  const tags = parseTags(tagsInput.value);

  previewTitle.textContent = title || "等待输入标题";
  previewBody.textContent = body || "在左侧输入正文后，这里会展示内容预览。";
  wordCount.textContent = `${body.length} 字`;

  previewTags.replaceChildren(
    ...tags.map((tag) => {
      const tagElement = document.createElement("span");
      tagElement.className = "tag";
      tagElement.textContent = `#${tag}`;
      return tagElement;
    }),
  );
}

[titleInput, bodyInput, tagsInput].forEach((input) => {
  input.addEventListener("input", renderPreview);
});

renderPreview();
