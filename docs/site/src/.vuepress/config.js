const { description } = require('../../package')

enSideBar = {
  nav: [
    {
      text: 'docs',
      link: '/docs/getting-started/introduction',
    },
    {
      text: 'github',
      link: 'https://github.com/sealerio/sealer'
    }
  ],
  sidebar: {
    '/docs/': [
      {
        title: 'Getting Started',
        collapsable: true,
        children: [
          'getting-started/introduction',
          'getting-started/quick-start',
          'getting-started/run-cloudimage',
          'getting-started/using-clusterfile',
          'getting-started/build-cloudimage',
          'getting-started/build-appimage',
          'getting-started/config',
          'getting-started/plugin',
          'getting-started/applications',
        ]
      },
      {
        title: 'Advanced',
        collapsable: true,
        children: [
          'advanced/architecture',
          'advanced/arm-cloudimage',
          'advanced/containerd-baseimage',
          'advanced/define-cloudimage',
          'advanced/develop-plugin',
          'advanced/gpu-cloudimage',
          'advanced/raw-docker-baseimage',
          'advanced/registry-configuration',
          'advanced/save-charts-package',
          'advanced/takeover-existed-cluster',
          'advanced/use-kyverno-baseimage',
          'advanced/use-sealer-in-container',
        ]
      },
      {
        title: 'Reference',
        collapsable: true,
        children: [
          'reference/cli',
          'reference/cloudrootfs',
          'reference/clusterfile',
          'reference/kubefile',
        ]
      },
      {
        title: 'Contributing',
        collapsable: true,
        children: [
          'contributing/code-of-conduct',
          'contributing/contribute',
        ]
      },
      {
        title: 'Help',
        collapsable: true,
        children: [
          'help/contact',
          'help/faq',
        ]
      },
    ],
  },
};

zhSideBar = {
  selectText:'选择语言',
  nav: [
    {
      text: '文档',
      link: '/zh/getting-started/introduction',
    },
    {
      text: 'github',
      link: 'https://github.com/sealerio/sealer'
    }
  ],
  sidebar: {
    '/zh/': [
      {
        title: '快速开始',
        collapsable: true,
        children: [
          'getting-started/introduction',
          'getting-started/quick-start',
          'getting-started/using-clusterfile',
          'getting-started/use-cloudimage',
          'getting-started/build-cloudimage',
          'getting-started/build-appimage',
          'getting-started/config',
          'getting-started/plugin',
          'getting-started/applications',
        ]
      },
      {
        title: '高阶教程',
        collapsable: true,
        children: [
          'advanced/architecture',
          'advanced/arm-cloudimage',
          'advanced/containerd-baseimage',
          'advanced/define-cloudimage',
          'advanced/develop-plugin',
          'advanced/gpu-cloudimage',
          'advanced/raw-docker-baseimage',
          'advanced/registry-configuration',
          'advanced/save-charts-package',
          'advanced/use-kyverno-baseimage',
        ]
      },
      {
        title: 'CLI&API',
        collapsable: true,
        children: [
          'reference/cli',
          'reference/cloudrootfs',
          'reference/clusterfile',
          'reference/kubefile',
        ]
      },
      {
        title: '贡献',
        collapsable: true,
        children: [
          'contributing/code-of-conduct',
          'contributing/contribute',
        ]
      },
      {
        title: '帮助',
        collapsable: true,
        children: [
          'help/contact',
          'help/faq',
        ]
      },
    ],
  },
};

module.exports = {
  /**
   * Ref：https://v1.vuepress.vuejs.org/config/#title
   */
  title: 'sealer',
  /**
   * Ref：https://v1.vuepress.vuejs.org/config/#description
   */
  description: description,

  /**
   * Extra tags to be injected to the page HTML `<head>`
   *'
   * ref：https://v1.vuepress.vuejs.org/config/#head
   */
  head: [
    ['meta', { name: 'theme-color', content: '#3eaf7c' }],
    ['meta', { name: 'apple-mobile-web-app-capable', content: 'yes' }],
    ['meta', { name: 'apple-mobile-web-app-status-bar-style', content: 'black' }],
    ['link', { rel: 'icon', href: 'https://user-images.githubusercontent.com/8912557/139633211-96844d27-55d7-44a9-9cdc-5aea96441613.png' }]
  ],
  locales: {
    '/': {
      lang: 'en-US',
      title: 'sealer',
      description: description,
    },
    '/zh/': {
        lang: '简体中文',
        title: 'sealer',
        description: 'sealer 官方文档',
    }
  },

  /**
   * Theme configuration, here is the default theme configuration for VuePress.
   *
   * ref：https://v1.vuepress.vuejs.org/theme/default-theme-config.html
   */
  themeConfig: {
    repo: '',
    logo: 'https://user-images.githubusercontent.com/8912557/139633211-96844d27-55d7-44a9-9cdc-5aea96441613.png',
    editLinks: false,
    docsDir: '',
    editLinkText: '',
    lastUpdated: false,
    locales: {
      '/zh/': zhSideBar,
      '/': enSideBar,
    }
  },

  /**
   * Apply plugins, ref：https://v1.vuepress.vuejs.org/zh/plugin/
   */
  plugins: [
    '@vuepress/plugin-back-to-top',
    '@vuepress/plugin-medium-zoom',
  ]
}
