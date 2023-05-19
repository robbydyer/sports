import React from 'react';
import Button from 'react-bootstrap/Button';
import Container from 'react-bootstrap/Container';
import Row from 'react-bootstrap/Row';
import Col from 'react-bootstrap/Col';
import Form from 'react-bootstrap/Form';
import Spinner from 'react-bootstrap/Spinner';
import { MatrixPostRet } from './util.js';
import { SetAllReq, LiveOnlyReq, Status } from './sportsmatrix/sportsmatrix_pb';
import 'bootstrap/dist/css/bootstrap.min.css';

function jsonToStatus(jsonDat) {
    var dat = JSON.parse(jsonDat)
    var status = new Status();
    status.setScreenOn(dat.screen_on);
    status.setWebboardOn(dat.webboard_on);

    return status;
}

class Home extends React.Component {
    constructor(props) {
        super(props);
        var status = new Status();
        this.state = {
            "status": status,
            "loading": false,
        };
    }
    async componentDidMount() {
        await this.getStatus();
    }

    getStatus = async () => {
        await MatrixPostRet("matrix.v1.Sportsmatrix/GetStatus", '{}').then((resp) => {
            if (resp.ok) {
                return resp.text();
            }
            throw resp;
        }).then((data) => {
            var dat = jsonToStatus(data);
            this.setState({
                "status": dat,
            })
        })
    }


    updateStatus = async () => {
        var req = this.state.status;
        await MatrixPostRet("matrix.v1.Sportsmatrix/SetStatus", JSON.stringify(req.toObject()));
        await this.getStatus();
    }

    handleLiveOnlySwitch = async (switchState) => {
        var req = new LiveOnlyReq();
        req.setLiveOnly(switchState)
        await MatrixPostRet("matrix.v1.Sportsmatrix/SetLiveOnly", JSON.stringify(req.toObject()));
        this.props.doSync();
    }

    disableAll = async () => {
        var req = new SetAllReq();
        req.setEnabled(false)
        await MatrixPostRet("matrix.v1.Sportsmatrix/SetAll", JSON.stringify(req.toObject()));
        this.props.doSync();
    }
    enableAll = async () => {
        var req = new SetAllReq();
        req.setEnabled(true)
        await MatrixPostRet("matrix.v1.Sportsmatrix/SetAll", JSON.stringify(req.toObject()));
        this.props.doSync();
    }

    restartMatrix = async () => {
        await MatrixPostRet("matrix.v1.Sportsmatrix/RestartService", '{}');
        this.setState({
            "loading": true,
        })
        setTimeout(() => {
            this.props.doSync();
            this.setState({
                "loading": false,
            })
        }, 10000);
    }

    nextBoard = () => {
        MatrixPostRet("matrix.v1.Sportsmatrix/NextBoard", '{}')
    }

    render() {
        return (
            <Container fluid>
                <Row className="text-left">
                    <Col>
                        <Form.Switch id="screen" label="Screen On/Off" checked={this.state.status.getScreenOn()}
                            onChange={() => { this.state.status.setScreenOn(!this.state.status.getScreenOn()); this.updateStatus(); }} />
                    </Col>
                </Row>
                <Row className="text-left">
                    <Col>
                        <Form.Switch id="webboard" label="Web Board On/Off" checked={this.state.status.getWebboardOn()}
                            onChange={() => { this.state.status.setWebboardOn(!this.state.status.getWebboardOn()); this.updateStatus(); }} />
                    </Col>
                </Row>
                <Row className="text-left">
                    <Col>
                        <Button variant="primary" onClick={this.nextBoard}>Next Board</Button>
                    </Col>
                </Row>
                <Row>
                    <Col>
                        <Button variant="primary" onClick={() => this.handleLiveOnlySwitch(true)}>Live Only</Button>
                    </Col>
                    <Col>
                        <Button variant="primary" onClick={() => this.handleLiveOnlySwitch(false)}>All Games</Button>
                    </Col>
                </Row>
                <Row className="text-left">
                    <Col>
                        <Button variant="primary" onClick={this.enableAll}>Enable All</Button>
                    </Col>
                    <Col>
                        <Button variant="primary" onClick={this.disableAll}>Disable All</Button>
                    </Col>
                </Row>
                <Row className="text-left">
                    <Col>
                        <Button variant="danger" onClick={this.restartMatrix} disabled={this.state.loading}>
                            {this.state.loading &&
                                <Spinner
                                    as="span"
                                    animation="border"
                                    size="sm"
                                    role="status"
                                />}
                            Restart Matrix Service
                        </Button>
                    </Col>
                </Row>
            </Container >
        );
    }
}
export default Home;