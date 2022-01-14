import React from 'react';
import Button from 'react-bootstrap/Button';
import Container from 'react-bootstrap/Container';
import Row from 'react-bootstrap/Row';
import Col from 'react-bootstrap/Col';
import Form from 'react-bootstrap/Form';
import Spinner from 'react-bootstrap/Spinner';
import { MatrixPostRet } from './util.js';
import { SetAllReq } from './sportsmatrix/sportsmatrix_pb';
import 'bootstrap/dist/css/bootstrap.min.css';

class Home extends React.Component {
    constructor(props) {
        super(props);
        this.state = {
            "status": {},
            "loading": false,
        };
    }
    async componentDidMount() {
        await this.getStatus();
    }

    getStatus = async () => {
        await MatrixPostRet("matrix.v1.Sportsmatrix/Status", '{}').then((resp) => {
            if (resp.ok) {
                return resp.text()
            }
            throw resp
        }).then((data) => {
            console.log("Got Home MatrixPostRet", data)
            this.setState({
                "status": JSON.parse(data),
            })
        });
    }

    handleSwitch = async (stateVar) => {
        var currentState = this.state.status[stateVar]
        if (currentState) {
            await MatrixPostRet("matrix.v1.Sportsmatrix/ScreenOff", '{}')
        } else {
            await MatrixPostRet("matrix.v1.Sportsmatrix/ScreenOn", '{}')
        }
        this.setState(prev => ({
            "status": {
                [stateVar]: !prev.status[stateVar],
            }
        }))
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
                        <Form.Switch id="screen" label="Screen On/Off" checked={this.state.status["screen_on"]} onChange={() => this.handleSwitch("screen_on")} />
                    </Col>
                </Row>
                <Row className="text-left">
                    <Col>
                        <Form.Switch id="webboard" label="Web Board On/Off" checked={this.state.status["webboard_on"]} onChange={() => this.handleSwitch("webboard_on")} />
                    </Col>
                </Row>
                <Row className="text-left">
                    <Col>
                        <Button variant="primary" onClick={this.nextBoard}>Next Board</Button>
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
                <Row>
                    <Col>
                        <Button variant="primary" onClick={() => this.handleLiveOnlySwitch(true)}>Live Games Only</Button>
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