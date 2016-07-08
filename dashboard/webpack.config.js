var glob = require('glob');

module.exports = {
  entry: glob.sync('./js/*.jsx'),
  output: {
    path: __dirname,
    filename: "bundle.js"
  },
  module: {
    loaders: [
      {
        test: /\.jsx?$/,
        loader: 'babel',
        query: {presets: ['react', 'es2015']}
      }
    ]
  }
};
