import React, { useState } from 'react';
import 'bootstrap/dist/css/bootstrap.min.css';
import Home from './Home.js';
import Sport from './Sport.js';
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
        </Router>
      </>
    );
  }
}
export default App;