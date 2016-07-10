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
        loader: 'babel',
        query: {presets: ['react', 'es2015']}
      }
    ]
  }
};
