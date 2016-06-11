const Router = ReactRouter.Router;
const Route = ReactRouter.Route;
const IndexRoute = ReactRouter.IndexRoute;
const Link = ReactRouter.Link;

class ItemListItem extends React.Component {
  render() {
    let info = this.props.info;

    return (
      <div className="ui items">
        <div className="item">
          <div className="image">
            <img src={info.image_url} />
            </div>
            <div className="content">
              <a className="header">{info.title}</a>
              <div className="meta">
                <span>{info.summary}</span>
              </div>
              <div className="description">
                <p></p>
              </div>
              <div className="extra">
                Published: {info.publication_time}
              </div>
            </div>
          </div>
        </div>
      );
    }
  }

class Feed extends React.Component {
  constructor() {
    super(...arguments);
    this.state = {
      info: {},
      items: [],
    };
  }
  componentWillMount() {
    let id = this.props.params.feedId;

    fetch(`/api/feeds/${id}`)
    .then(resp => resp.json())
    .then(data => this.setState(_.assign(this.state, {info: data})));

    fetch(`/api/feeds/${id}/items`)
    .then(resp => resp.json())
    .then(data => this.setState(_.assign(this.state, {items: data})));
  }

  render() {
    let items = this.state.items
    .sort((a, b) => a.publication_time < b.publication_time ? 1 : -1)
    .map((item) => {
      return (
        <ItemListItem key={item.id} info={item} />
      );
    });

    let info = this.state.info;
    return (
      <div>
        <h1>{info.title}</h1>
        {items}
      </div>
    );
  }
}

class FeedListItem extends React.Component {
  render() {
    var info = this.props.feedInfo;

    return (
      <div className="item">
        <div className="content">
          <Link to={`/feeds/${info.id}`} className="header">{info.title}</Link>
          <div className="description">
            {info.url} | {info.items.length} items
          </div>
        </div>
      </div>
    );
  }
}

FeedListItem.propTypes = {
  feedInfo: React.PropTypes.object.isRequired,
};

class FeedList extends React.Component {
  constructor() {
    super(...arguments);
    this.state = {
      feeds: [],
    };
  }

  componentWillMount() {
    fetch('/api/feeds')
    .then((resp) => resp.json())
    .then((data) => this.setState({feeds: data}));
  }

  render() {
    var feeds = this.state.feeds.map((feed) => {
      return (
        <FeedListItem key={feed.id} feedInfo={feed} />
      );
    });

    return (
      <div className="ui relaxed divided list">
        {feeds}
      </div>
    );
  }
}

ReactDOM.render((
  <Router>
    <Route path="/" component={FeedList} />
    <IndexRoute component={FeedList} />
    <Route path="feeds/:feedId" component={Feed} />
  </Router>
), document.getElementById('content'));
