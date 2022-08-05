import React from 'react';
import 'bootstrap/dist/css/bootstrap.min.css';
import Button from 'react-bootstrap/Button';
import Container from 'react-bootstrap/Container';
import Row from 'react-bootstrap/Row';
import Col from 'react-bootstrap/Col';
import Form from 'react-bootstrap/Form';
import Image from 'react-bootstrap/Image';
import { MatrixPostRet, JumpToBoard } from './util';
import { SetStatusReq, Status } from './racingboard/racingboard_pb';
import { LogoSrc } from './Logo';


function jsonToStatus(jsonDat) {
    var d = JSON.parse(jsonDat);
    var dat = d.status;
    var status = new Status();
    status.setEnabled(dat.enabled);
    status.setScrollEnabled(dat.scroll_enabled);

    return status;
}

class Sport extends React.Component {
    constructor(props) {
        super(props);
        var status = new Status();
        var visible = "hidden"
        if (this.props.withImg) {
            visible = "visible"
        }
        this.state = {
            "status": status,
            "imgVisible": visible,
        };
    }
    async componentDidMount() {
        await this.getStatus()
    }
    getStatus = async () => {
        await MatrixPostRet(this.props.sport + "/racing.v1.Racing/GetStatus", '{}').then((resp) => {
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
        var req = new SetStatusReq();
        req.setStatus(this.state.status);
        await MatrixPostRet(this.props.sport + "/racing.v1.Racing/SetStatus", JSON.stringify(req.toObject()));
        await this.getStatus();
    }

    doJump = async () => {
        await JumpToBoard(this.props.sport);
        console.log("Syncing from racing")
        this.props.doSync();
    }

    render() {
        return (
            <Container fluid>
                <Row className="text-center">
                    <Col>
                        <Image src={LogoSrc(this.props.sport)} style={{ height: '100px', width: 'auto', visibility: this.state.imgVisible }} fluid />
                    </Col>
                </Row>
                <Row className="text-left">
                    <Col>
                        <Form.Switch id={this.props.sport + "enabler"} label="Enable/Disable" checked={this.state.status.getEnabled()}
                            onChange={() => { this.state.status.setEnabled(!this.state.status.getEnabled()); this.updateStatus(); }} />
                    </Col>
                </Row>
                <Row className="text-left">
                    <Col>
                        <Form.Switch id={this.props.sport + "scroller"} label="Scroll Mode" checked={this.state.status.getScrollEnabled()}
                            onChange={() => { this.state.status.setScrollEnabled(!this.state.status.getScrollEnabled()); this.updateStatus(); }} />
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

export default Sport;