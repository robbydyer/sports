import React from 'react';
import 'bootstrap/dist/css/bootstrap.min.css';
import Sport from './Sport.js';
import ImageBoard from './ImageBoard.js';
import Board from './Board.js';
import Weather from './Weather.js';
import TopNav from './Nav.js';
import All from './All.js';
import BasicBoard from './BasicBoard';
import { BrowserRouter as Router, Route } from 'react-router-dom';
import SwaggerUI from 'swagger-ui-react';
import "swagger-ui-react/swagger-ui.css";
import swag from './matrix.swagger.json';

class App extends React.Component {
  render() {
    var sports = ["ncaaf", "nhl", "mlb", "ncaam", "nfl", "nba", "mls", "epl"].map((sport) =>
      <Route path={"/" + sport} render={() => {
        <Sport sport={sport} id={sport} key={sport} />
      }} />
    );
    return (
      <>
        <Router>
          <TopNav />
          {sports}
          <Route path="/" exact component={All} />
          <Route path="/pga" render={() => <BasicBoard id="pga" name="pga" key="pga" path="stat/pga" />} />
          <Route path="/img" exact component={ImageBoard} />
          <Route path="/clock" render={() => <BasicBoard id="clock" name="clock" key="clock" />} />
          <Route path="/sys" render={() => <BasicBoard id="sys" name="sys" key="sys" />} />
          <Route path="/stocks" render={() => <BasicBoard id="stocks" name="stocks" key="stocks" />} />
          <Route path="/weather" exact component={Weather} />
          <Route path="/board" exact component={Board} />
          <Route path="/docs" exact component={() => <SwaggerUI spec={swag} />} />
        </Router>
        <hr />
      </>
    );
  }
}
export default App;