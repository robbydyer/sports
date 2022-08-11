import React from 'react';
import 'bootstrap/dist/css/bootstrap.min.css';
import Button from 'react-bootstrap/Button';
import Container from 'react-bootstrap/Container';
import Row from 'react-bootstrap/Row';
import Col from 'react-bootstrap/Col';
import Image from 'react-bootstrap/Image';
import Form from 'react-bootstrap/Form';
import { MatrixPostRet, JumpToBoard } from './util';
import * as pb from './weatherboard/weatherboard_pb';
import { LogoSrc } from './Logo';

function jsonToStatus(jsonDat) {
    var d = JSON.parse(jsonDat);
    var dat = d.status;
    var status = new pb.Status();
    status.setEnabled(dat.enabled);
    status.setScrollEnabled(dat.scroll_enabled);
    status.setDailyEnabled(dat.daily_enabled);
    status.setHourlyEnabled(dat.hourly_enabled);

    return status;
}

class Weather extends React.Component {
    constructor(props) {
        super(props);
        this.state = {
            "status": new pb.Status(),
        };
    }
    async componentDidMount() {
        await this.getStatus();
    }
    getStatus = async () => {
        await MatrixPostRet("weather.v1.WeatherBoard/GetStatus", '{}').then((resp) => {
            if (resp.ok) {
                return resp.text()
            }
            throw resp
        }).then((data) => {
            var dat = jsonToStatus(data);
            this.setState({
                "status": dat,
            })
        });
    }

    updateStatus = async () => {
        var req = new pb.SetStatusReq();
        req.setStatus(this.state.status);
        await MatrixPostRet("weather.v1.WeatherBoard/SetStatus", JSON.stringify(req.toObject()));
        await this.getStatus();
    }

    doJump = async () => {
        await JumpToBoard("weather");
        this.props.doSync();
    }

    render() {
        var img = (
            <Row className="text-center"><Col><Image src={LogoSrc("weather")} style={{ height: '100px', width: 'auto' }} fluid /></Col></Row>
        )
        return (
            <Container fluid>
                {this.props.withImg ? img : ""}
                <Row className="text-left">
                    <Col>
                        <Form.Switch id="weatherenabler" label="Enable/Disable" checked={this.state.status.getEnabled()}
                            onChange={() => { this.state.status.setEnabled(!this.state.status.getEnabled()); this.updateStatus(); }} />
                    </Col>
                </Row>
                <Row className="text-left">
                    <Col>
                        <Form.Switch id="weatherscroller" label="Scroll Mode" checked={this.state.status.getScrollEnabled()}
                            onChange={() => { this.state.status.setScrollEnabled(!this.state.status.getScrollEnabled()); this.updateStatus(); }} />
                    </Col>
                </Row>
                <Row className="text-left">
                    <Col>
                        <Form.Switch id="dailyenabler" label="Daily Forecast" checked={this.state.status.getDailyEnabled()}
                            onChange={() => { this.state.status.setDailyEnabled(!this.state.status.getDailyEnabled()); this.updateStatus(); }} />
                    </Col>
                </Row>
                <Row className="text-left">
                    <Col>
                        <Form.Switch id="hourlyenabler" label="Hourly Forecast" checked={this.state.status.getHourlyEnabled()}
                            onChange={() => { this.state.status.setHourlyEnabled(!this.state.status.getHourlyEnabled()); this.updateStatus(); }} />
                    </Col>
                </Row>
                <Row className="text-left">
                    <Col>
                        <Button variant="primary" onClick={() => { this.doJump(); }}>Jump</Button>
                    </Col>
                </Row>
            </Container>
        )
    }
}

export default Weather;