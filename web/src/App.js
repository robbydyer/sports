import React from 'react';
import 'bootstrap/dist/css/bootstrap.min.css';
import Home from './Home.js';
import Sport from './Sport.js';
import ImageBoard from './ImageBoard.js';
import Clock from './Clock.js';
import TopNav from './Nav.js';
import { BrowserRouter as Router, Route } from 'react-router-dom';

const MATRIX = "http://matrix.local:8080"

class App extends React.Component {
  screenOn() {
    console.log("Turning screen on")
    fetch(`${MATRIX}/screenon`, {
      method: "GET",
      mode: "cors",
    });
  }
  screenOff() {
    console.log("Turning screen off")
    fetch(`${MATRIX}/screenoff`, {
      method: "GET",
      mode: "cors",
    });
  }
  render() {
    return (
      <>
        <Router>
          <TopNav />
          <Route path="/" exact component={Home} />
          <Route path="/nhl" render={() => <Sport sport="nhl" />} />
          <Route path="/mlb" render={() => <Sport sport="mlb" />} />
          <Route path="/img" exact component={ImageBoard} />
          <Route path="/clock" exact component={Clock} />
        </Router>
      </>
    );
  }
}
export default App;