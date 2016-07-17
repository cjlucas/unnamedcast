module.exports = {
  entry: './js/app.jsx',
  output: {
    path: __dirname,
    filename: "bundle.js"
  },
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
