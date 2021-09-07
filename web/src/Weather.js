import React from 'react';
import 'bootstrap/dist/css/bootstrap.min.css';
import Button from 'react-bootstrap/Button';
import Container from 'react-bootstrap/Container';
import Row from 'react-bootstrap/Row';
import Col from 'react-bootstrap/Col';
import Image from 'react-bootstrap/Image';
import weatherimg from './weather.png';
import Form from 'react-bootstrap/Form';
import { GetStatus, CallMatrix, MatrixPost } from './util';

class Weather extends React.Component {
    constructor(props) {
        super(props);
        this.state = {
            "enabled": false,
            "scroll": false,
            "daily": false,
            "hourly": false,
        };
    }
    async componentDidMount() {
        this.updateStatus()
    }
    async updateStatus() {
        await GetStatus("weather/status", (val) => {
            this.setState({ "enabled": val })
        })
        await GetStatus(`weather/scrollstatus`, (val) => {
            this.setState({
                "scroll": val,
            })
        })
        await GetStatus("weather/dailystatus", (val) => {
            this.setState({ "daily": val })
        })
        await GetStatus("weather/hourlystatus", (val) => {
            this.setState({ "hourly": val })
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
    handleJump = (board) => {
        MatrixPost("jump", `{"board":"${board}"}`)
        this.updateStatus()
    }

    render() {
        return (
            <Container fluid>
                <Row className="text-center"><Col><Image src={weatherimg} style={{ height: '100px', width: 'auto' }} fluid /></Col></Row>
                <Row className="text-left">
                    <Col>
                        <Form.Switch id="weatherenabler" label="Enable/Disable" checked={this.state.enabled}
                            onChange={() => this.handleSwitch("weather/enable", "weather/disable", "enabled")} />
                    </Col>
                </Row>
                <Row className="text-left">
                    <Col>
                        <Form.Switch id="weatherscroller" label="Scroll Mode" checked={this.state.scroll}
                            onChange={() => this.handleSwitch(`weather/scrollon`, `weather/scrolloff`, "scroll")} />
                    </Col>
                </Row>
                <Row className="text-left">
                    <Col>
                        <Form.Switch id="dailyenabler" label="Daily Forecast" checked={this.state.daily}
                            onChange={() => this.handleSwitch("weather/dailyenable", "weather/dailydisable", "daily")} />
                    </Col>
                </Row>
                <Row className="text-left">
                    <Col>
                        <Form.Switch id="hourlyenabler" label="Hourly Forecast" checked={this.state.hourly}
                            onChange={() => this.handleSwitch("weather/hourlyenable", "weather/hourlydisable", "hourly")} />
                    </Col>
                </Row>
                <Row className="text-left">
                    <Col>
                        <Button variant="primary" onClick={() => this.handleJump("weather")}>Jump</Button>
                    </Col>
                </Row>
            </Container>
        )
    }
}

export default Weather;