var HtmlWebpackPlugin = require('html-webpack-plugin');

module.exports = {
  entry: './js/app.jsx',
  output: {
    path: 'dist',
    filename: "bundle.js"
  },
  plugins: [
      new HtmlWebpackPlugin({  // Also generate a test.html
      template: 'index.ejs'
    })
  ],
  module: {
    loaders: [
      {
        test: /\.jsx?$/,
        exclude: /(node_modules)/,
        loader: 'babel',
        query: {
          presets: ['react'],
          plugins: ['transform-es2015-modules-commonjs'],
        }
      }
    ]
  }
};
