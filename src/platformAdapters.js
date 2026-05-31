const PLATFORM_ADAPTERS = [
  {
    id: "wechat",
    name: "公众号",
    tone: "结构清晰、适合长文阅读",
    adapt(content) {
      return {
        platformId: "wechat",
        platformName: "公众号",
        title: content.title,
        body: buildWechatBody(content),
        tags: content.tags.slice(0, 3),
        tips: ["首段突出价值点", "正文保留分段", "文末适合放关注引导"],
      };
    },
  },
  {
    id: "zhihu",
    name: "知乎",
    tone: "问题导向、强调观点和论证",
    adapt(content) {
      return {
        platformId: "zhihu",
        platformName: "知乎",
        title: `${content.title}：我的方法和思考`,
        body: buildZhihuBody(content),
        tags: content.tags.slice(0, 5),
        tips: ["标题适合带问题意识", "正文突出经验和理由", "结尾可补充讨论问题"],
      };
    },
  },
  {
    id: "bilibili",
    name: "B站",
    tone: "轻松直接、适合动态或专栏引流",
    adapt(content) {
      return {
        platformId: "bilibili",
        platformName: "B站",
        title: `【创作分享】${content.title}`,
        body: buildBilibiliBody(content),
        tags: content.tags.slice(0, 6),
        tips: ["标题可以更有记忆点", "正文适合加入互动引导", "标签可覆盖内容领域和受众"],
      };
    },
  },
  {
    id: "xiaohongshu",
    name: "小红书",
    tone: "种草笔记、强调场景和关键词",
    adapt(content) {
      return {
        platformId: "xiaohongshu",
        platformName: "小红书",
        title: `${content.title}｜实用分享`,
        body: buildXiaohongshuBody(content),
        tags: content.tags.slice(0, 8),
        tips: ["开头给出使用场景", "正文适合短句和要点", "标签建议覆盖搜索关键词"],
      };
    },
  },
];

function createSourceContent({ title, body, tags }) {
  return {
    title: title.trim() || "未命名内容",
    body: body.trim(),
    tags: tags.map((tag) => tag.trim()).filter(Boolean),
  };
}

function adaptContentForPlatforms(content) {
  return PLATFORM_ADAPTERS.map((adapter) => adapter.adapt(content));
}

function buildWechatBody(content) {
  return [
    `导语：${content.title}`,
    "",
    content.body || "这里是正文内容，可继续补充完整观点、案例和结论。",
    "",
    "如果这篇内容对你有帮助，欢迎收藏并持续关注后续更新。",
  ].join("\n");
}

function buildZhihuBody(content) {
  return [
    `先说结论：${content.title}的关键在于把内容拆成可复用的结构。`,
    "",
    content.body || "这里可以补充背景、分析过程和个人经验。",
    "",
    "你也可以根据自己的平台和受众，继续调整表达方式。",
  ].join("\n");
}

function buildBilibiliBody(content) {
  return [
    `${content.title}`,
    "",
    content.body || "这是一条适合 B 站动态或专栏的内容简介。",
    "",
    "欢迎在评论区交流你的做法。",
  ].join("\n");
}

function buildXiaohongshuBody(content) {
  const tagText = content.tags.map((tag) => `#${tag}`).join(" ");

  return [
    `${content.title}`,
    "",
    content.body || "今天分享一个适合创作者提升发布效率的小方法。",
    "",
    tagText,
  ]
    .filter(Boolean)
    .join("\n");
}
