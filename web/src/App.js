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
          <Route path="/mlb" render={() => <Sport sport="mlb" id="mlb" key="mlb" />} />
          <Route path="/ncaaf" render={() => <Sport sport="ncaaf" id="ncaaf" key="ncaaf" />} />
          <Route path="/nhl" render={() => <Sport sport="nhl" id="nhl" key="nhl" />} />
          <Route path="/ncaam" render={() => <Sport sport="ncaam" id="ncaam" key="ncaam" />} />
          <Route path="/nfl" render={() => <Sport sport="nfl" id="nfl" key="nfl" />} />
          <Route path="/nba" render={() => <Sport sport="nba" id="nba" key="nba" />} />
          <Route path="/mls" render={() => <Sport sport="mls" id="mls" key="mls" />} />
          <Route path="/epl" render={() => <Sport sport="epl" id="epl" key="epl" />} />
          <Route path="/pga" render={() => <BasicBoard id="pga" name="pga" key="pga" path="stat/pga" />} />
          <Route path="/img" exact component={ImageBoard} />
          <Route path="/clock" render={() => <BasicBoard id="clock" name="clock" key="clock" />} />
          <Route path="/sys" render={() => <BasicBoard id="sys" name="sys" key="sys" />} />
          <Route path="/stocks" render={() => <BasicBoard id="stocks" name="stocks" key="stocks" />} />
          <Route path="/gcal" render={() => <BasicBoard id="gcal" name="gcal" key="gcal" />} />
          <Route path="/weather" exact component={Weather} />
          <Route path="/board" exact component={Board} />
          <Route path="/docs" exact component={() => <SwaggerUI spec={swag} />} />
          <Route path="/f1" exact component={() => <Racing sport="f1" id="f1" key="f1" />} />
          <Route path="/irl" exact component={() => <Racing sport="irl" id="irl" key="irl" />} />
        </Router>
        <hr />
      </>
    );
  }
}
export default App;