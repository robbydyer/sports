import React from 'react';
import 'bootstrap/dist/css/bootstrap.min.css';
import Home from './Home.js';
import Sport from './Sport.js';
import Pga from './Pga.js';
import ImageBoard from './ImageBoard.js';
import Clock from './Clock.js';
import Board from './Board.js';
import Sys from './Sys.js';
import Stocks from './Stocks.js';
import TopNav from './Nav.js';
import All from './All.js';
import { BrowserRouter as Router, Route } from 'react-router-dom';

var BACKEND = "http://" + window.location.host

class App extends React.Component {
  screenOn() {
    console.log("Turning screen on")
    fetch(`${BACKEND}/api/screenon`, {
      method: "GET",
      mode: "cors",
    });
  }
  screenOff() {
    console.log("Turning screen off")
    fetch(`${BACKEND}/api/screenoff`, {
      method: "GET",
      mode: "cors",
    });
  }

  render() {
    return (
      <>
        <Router>
          <TopNav />
          <Route path="/" exact component={All} />
          <Route path="/nhl" render={() => <Sport sport="nhl" />} />
          <Route path="/mlb" render={() => <Sport sport="mlb" />} />
          <Route path="/pga" render={() => <Pga />} />
          <Route path="/ncaam" render={() => <Sport sport="ncaam" />} />
          <Route path="/ncaaf" render={() => <Sport sport="ncaaf" />} />
          <Route path="/nba" render={() => <Sport sport="nba" />} />
          <Route path="/nfl" render={() => <Sport sport="nfl" />} />
          <Route path="/mls" render={() => <Sport sport="mls" />} />
          <Route path="/epl" render={() => <Sport sport="epl" />} />
          <Route path="/img" exact component={ImageBoard} />
          <Route path="/clock" exact component={Clock} />
          <Route path="/sys" exact component={Sys} />
          <Route path="/stocks" exact component={Stocks} />
          <Route path="/board" exact component={Board} />
        </Router>
        <hr />
      </>
    );
  }
}
export default App;