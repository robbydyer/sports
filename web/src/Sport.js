import React from 'react';
import 'bootstrap/dist/css/bootstrap.min.css';
import Button from 'react-bootstrap/Button';
import Container from 'react-bootstrap/Container';
import Row from 'react-bootstrap/Row';
import Col from 'react-bootstrap/Col';
import Form from 'react-bootstrap/Form';
import Image from 'react-bootstrap/Image';
import nhllogo from './nhllogo.jpeg';
import mlblogo from './mlb.png';
import ncaamlogo from './ncaam.png';
import nbalogo from './nba.png';
import nfllogo from './nfl.png';
import mlslogo from './mls.png';
import ncaaflogo from './ncaaf.png'
import epllogo from './epl.png'
import { GetStatus, CallMatrix } from './util';

class Sport extends React.Component {
    constructor(props) {
        super(props);
        this.state = {
            "enabled": false,
            "hideFavorite": false,
            "stickyFavorite": false,
            "stats": false,
            "scroll": false,
            "statscroll": false,
            "tightscroll": false,
            "record": false,
            "odds": false,
        };
    }
    async componentDidMount() {
        await GetStatus(`${this.props.sport}/status`, (val) => {
            this.setState({
                "enabled": val,
            })
        })
        await GetStatus(`${this.props.sport}/scrollstatus`, (val) => {
            this.setState({
                "scroll": val,
            })
        })
        await GetStatus(`${this.props.sport}/tightscrollstatus`, (val) => {
            this.setState({
                "tightscroll": val,
            })
        })
        await GetStatus(`${this.props.sport}/stats/status`, (val) => {
            this.setState({
                "stats": val,
            })
        })
        await GetStatus(`${this.props.sport}/stats/scrollstatus`, (val) => {
            this.setState({
                "statscroll": val,
            })
        })
        await GetStatus(`${this.props.sport}/favoritescorestatus`, (val) => {
            this.setState({
                "hideFavorite": val,
            })
        })
        await GetStatus(`${this.props.sport}/favoritestickystatus`, (val) => {
            this.setState({
                "stickyFavorite": val,
            })
        })
        await GetStatus(`${this.props.sport}/oddsstatus`, (val) => {
            this.setState({
                "odds": val,
            })
        })
        await GetStatus(`${this.props.sport}/recordrankstatus`, (val) => {
            this.setState({
                "record": val,
            })
        })
    }

    handleSwitch = (apiOn, apiOff, stateVar) => {
        var currentState = this.state[stateVar]
        console.log("handle switch", currentState)
        if (currentState) {
            console.log("Turn off", apiOff)
            CallMatrix(apiOff);
        } else {
            console.log("Turn on", apiOn)
            CallMatrix(apiOn);
        }
        this.setState(prev => ({
            [stateVar]: !prev[stateVar],
        }))
    }

    logosrc() {
        if (this.props.sport == "nhl") {
            return nhllogo
        } else if (this.props.sport == "ncaam") {
            return ncaamlogo
        } else if (this.props.sport == "ncaaf") {
            return ncaaflogo
        } else if (this.props.sport == "nba") {
            return nbalogo
        } else if (this.props.sport == "nfl") {
            return nfllogo
        } else if (this.props.sport == "mls") {
            return mlslogo
        } else if (this.props.sport == "epl") {
            return epllogo
        } else {
            return mlblogo
        }
    }
    render() {
        return (
            <Container fluid>
                <Row className="text-center"><Col><Image src={this.logosrc()} style={{ height: '100px', width: 'auto' }} fluid /></Col></Row>
                <Row className="text-center">
                    <Col>
                        <Form.Switch id={this.props.sport + "enabler"} label="Enable/Disable" checked={this.state.enabled}
                            onChange={() => this.handleSwitch(`${this.props.sport}/enable`, `${this.props.sport}/disable`, "enabled")} />
                    </Col>
                </Row>
                <Row className="text-center">
                    <Col>
                        <Form.Switch id={this.props.sport + "scroller"} label="Scroll Mode" checked={this.state.scroll}
                            onChange={() => this.handleSwitch(`${this.props.sport}/scrollon`, `${this.props.sport}/scrolloff`, "scroll")} />
                    </Col>
                </Row>
                <Row className="text-center">
                    <Col>
                        <Form.Switch id={this.props.sport + "tightscroller"} label="Back-to-back Scroll Mode" checked={this.state.tightscroll}
                            onChange={() => this.handleSwitch(`${this.props.sport}/tightscrollon`, `${this.props.sport}/tightscrolloff`, "tightscroll")} />
                    </Col>
                </Row>
                <Row className="text-center">
                    <Col>
                        <Form.Switch id={this.props.sport + "stats"} label="Stats" checked={this.state.stats}
                            onChange={() => this.handleSwitch(`${this.props.sport}/stats/enable`, `${this.props.sport}/stats/disable`, "stats")} />
                    </Col>
                </Row>
                <Row className="text-center">
                    <Col>
                        <Form.Switch id={this.props.sports + "statscroll"} label="Stats Scroll Mode" checked={this.state.statscroll}
                            onChange={() => this.handleSwitch(`${this.props.sport}/stats/scrollon`, `${this.props.sport}/stats/scrolloff`, "statscroll")} />
                    </Col>
                </Row>
                <Row className="text-center">
                    <Col>
                        <Form.Switch id={this.props.sport + "favscore"} label="Hide Favorite Scores" checked={this.state.hideFavorite}
                            onChange={() => this.handleSwitch(`${this.props.sport}/hidefavoritescore`, `${this.props.sport}/showfavoritescore`, "hideFavorite")} />
                    </Col>
                </Row>
                <Row className="text-center">
                    <Col>
                        <Form.Switch id={this.props.sport + "odds"} label="Show Odds" checked={this.state.odds}
                            onChange={() => this.handleSwitch(`${this.props.sport}/oddson`, `${this.props.sport}/oddsoff`, "odds")} />
                    </Col>
                </Row>
                <Row className="text-center">
                    <Col>
                        <Form.Switch id={this.props.sport + "record"} label="Record + Rank" checked={this.state.record}
                            onChange={() => this.handleSwitch(`${this.props.sport}/recordrankon`, `${this.props.sport}/recordrankoff`, "record")} />
                    </Col>
                </Row>
                <Row className="text-center">
                    <Col>
                        <Form.Switch id={this.props.sport + "favstick"} label="Stick Favorite Live Games" checked={this.state.stickyFavorite}
                            onChange={() => this.handleSwitch(`${this.props.sport}/favoritesticky`, `${this.props.sport}/favoriteunstick`, "stickyFavorite")} />
                    </Col>
                </Row>
            </Container>
        )
    }
}

export default Sport;