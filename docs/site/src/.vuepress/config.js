const { description } = require('../../package')

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
   *
   * ref：https://v1.vuepress.vuejs.org/config/#head
   */
  head: [
    ['meta', { name: 'theme-color', content: '#3eaf7c' }],
    ['meta', { name: 'apple-mobile-web-app-capable', content: 'yes' }],
    ['meta', { name: 'apple-mobile-web-app-status-bar-style', content: 'black' }]
  ],

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
    nav: [
      {
        text: 'docs',
        link: '/docs/getting-started/introduction',
      },
      {
        text: 'github',
        link: 'https://github.com/alibaba/sealer'
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
            'getting-started/run-cluster',
            'getting-started/build-cloudimage',
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
            'advanced/define-cloudrootfs',
            'advanced/gpu-cloudimage',
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
    }
  },

  /**
   * Apply plugins，ref：https://v1.vuepress.vuejs.org/zh/plugin/
   */
  plugins: [
    '@vuepress/plugin-back-to-top',
    '@vuepress/plugin-medium-zoom',
  ]
}
