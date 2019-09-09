module.exports = {
  publicPath: '/ui',
  outputDir: 'dist',
  lintOnSave: false,
  chainWebpack: config => {
    config.plugins.delete('progress');
    config.plugin('simple-progress-webpack-plugin').use(require.resolve('simple-progress-webpack-plugin'), [
      {
        format: 'minimal',
      }
    ])
  },
};
