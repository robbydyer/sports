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
import { GetStatus, CallMatrix } from './util';

class Sport extends React.Component {
    constructor(props) {
        super(props);
        this.state = {
            "enabled": false,
            "hideFavorite": false,
            "stickyFavorite": false,
            "stats": false,
        };
    }
    async componentDidMount() {
        await GetStatus(`${this.props.sport}/status`, (val) => {
            this.setState({
                "enabled": val,
            })
        })
        await GetStatus(`${this.props.sport}/stats/status`, (val) => {
            this.setState({
                "stats": val,
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
        } else if (this.props.sport == "nba") {
            return nbalogo
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
                        <Form.Switch id="enabler" label="Enable/Disable" checked={this.state.enabled}
                            onChange={() => this.handleSwitch(`${this.props.sport}/enable`, `${this.props.sport}/disable`, "enabled")} />
                    </Col>
                </Row>
                <Row className="text-center">
                    <Col>
                        <Form.Switch id="stats" label="Stats" checked={this.state.stats}
                            onChange={() => this.handleSwitch(`${this.props.sport}/stats/enable`, `${this.props.sport}/stats/disable`, "stats")} />
                    </Col>
                </Row>
                <Row className="text-center">
                    <Col>
                        <Form.Switch id="favscore" label="Hide Favorite Scores" checked={this.state.hideFavorite}
                            onChange={() => this.handleSwitch(`${this.props.sport}/hidefavoritescore`, `${this.props.sport}/showfavoritescore`, "hideFavorite")} />
                    </Col>
                </Row>
                <Row className="text-center">
                    <Col>
                        <Form.Switch id="favscore" label="Stick Favorite Live Games" checked={this.state.stickyFavorite}
                            onChange={() => this.handleSwitch(`${this.props.sport}/favoritesticky`, `${this.props.sport}/favoriteunstick`, "stickyFavorite")} />
                    </Col>
                </Row>
            </Container>
        )
    }
}

export default Sport;