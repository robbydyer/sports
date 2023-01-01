import React from 'react';
import 'bootstrap/dist/css/bootstrap.min.css';
import Sport from './Sport.js';
import Racing from './Racing.js';
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
    return (
      <>
        <Router>
          <TopNav />
          <Route path="/" exact component={All} />
          <Route path="/mlb" render={() => <Sport sport="mlb" id="mlb" key="mlb" withImg="true" />} />
          <Route path="/ncaaf" render={() => <Sport sport="ncaaf" id="ncaaf" key="ncaaf" withImg="true" />} />
          <Route path="/nhl" render={() => <Sport sport="nhl" id="nhl" key="nhl" withImg="true" />} />
          <Route path="/ncaam" render={() => <Sport sport="ncaam" id="ncaam" key="ncaam" withImg="true" />} />
          <Route path="/nfl" render={() => <Sport sport="nfl" id="nfl" key="nfl" withImg="true" />} />
          <Route path="/nba" render={() => <Sport sport="nba" id="nba" key="nba" withImg="true" />} />
          <Route path="/mls" render={() => <Sport sport="mls" id="mls" key="mls" withImg="true" />} />
          <Route path="/epl" render={() => <Sport sport="epl" id="epl" key="epl" withImg="true" />} />
          <Route path="/dfl" render={() => <Sport sport="dfl" id="dfl" key="dfl" withImg="true" />} />
          <Route path="/dfb" render={() => <Sport sport="dfb" id="dfb" key="dfb" withImg="true" />} />
          <Route path="/uefa" render={() => <Sport sport="uefa" id="uefa" key="uefa" withImg="true" />} />
          <Route path="/fifa" render={() => <Sport sport="fifa" id="fifa" key="fifa" withImg="true" />} />
          <Route path="/pga" render={() => <BasicBoard id="pga" name="pga" key="pga" path="stat/pga" withImg="true" />} />
          <Route path="/img" render={() => <ImageBoard withImg="true" />} />
          <Route path="/clock" render={() => <BasicBoard id="clock" name="clock" key="clock" withImg="true" />} />
          <Route path="/sys" render={() => <BasicBoard id="sys" name="sys" key="sys" withImg="true" />} />
          <Route path="/stocks" render={() => <BasicBoard id="stocks" name="stocks" key="stocks" withImg="true" />} />
          <Route path="/gcal" render={() => <BasicBoard id="gcal" name="gcal" key="gcal" withImg="true" />} />
          <Route path="/weather" render={() => <Weather withImg="true" />} />
          <Route path="/board" exact component={Board} />
          <Route path="/docs" exact component={() => <SwaggerUI spec={swag} />} />
          <Route path="/f1" exact component={() => <Racing sport="f1" id="f1" key="f1" withImg="true" />} />
          <Route path="/irl" exact component={() => <Racing sport="irl" id="irl" key="irl" withImg="true" />} />
          <Route path="/ncaaw" render={() => <Sport sport="ncaaw" id="ncaaw" key="ncaaw" withImg="true" />} />
          <Route path="/wnba" render={() => <Sport sport="wnba" id="wnba" key="wnba" withImg="true" />} />
          <Route path="/ligue" render={() => <Sport sport="ligue" id="ligue" key="ligue" withImg="true" />} />
          <Route path="/seriea" render={() => <Sport sport="seriea" id="seriea" key="seriea" withImg="true" />} />
          <Route path="/laliga" render={() => <Sport sport="laliga" id="laliga" key="laliga" withImg="true" />} />
        </Router>
        <hr />
      </>
    );
  }
}
export default App;