import React from 'react';
import 'bootstrap/dist/css/bootstrap.min.css';
import Button from 'react-bootstrap/Button';
import Container from 'react-bootstrap/Container';
import Row from 'react-bootstrap/Row';
import Col from 'react-bootstrap/Col';
import Image from 'react-bootstrap/Image';
import imgimg from './image.png';
import Form from 'react-bootstrap/Form';
import { MatrixPostRet, JumpToBoard } from './util';
import * as pb from './imageboard/imageboard_pb';
import { LogoSrc } from './Logo';


function jsonToStatus(jsonDat) {
    var d = JSON.parse(jsonDat);
    var dat = d.status;
    var status = new pb.Status();
    status.setEnabled(dat.enabled);
    status.setDiskcacheEnabled(dat.diskcache_enabled);
    status.setMemcacheEnabled(dat.memcache_enabled);

    return status;
}

class ImageBoard extends React.Component {
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
        await MatrixPostRet("imageboard.v1.ImageBoard/GetStatus", '{}').then((resp) => {
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
        await MatrixPostRet("imageboard.v1.ImageBoard/SetStatus", JSON.stringify(req.toObject()));
        this.getStatus();
    }

    doJump = async () => {
        await JumpToBoard("img");
        this.props.doSync();
    }
    render() {
        var img = (
            <Row className="text-center"><Col><Image src={LogoSrc("img")} style={{ height: '100px', width: 'auto' }} fluid /></Col></Row>
        )
        return (
            <Container fluid>
                {this.props.withImg ? img : ""}
                <Row className="text-left">
                    <Col>
                        <Form.Switch id="imgenabler" label="Enable/Disable" checked={this.state.status.getEnabled()}
                            onChange={() => { this.state.status.setEnabled(!this.state.status.getEnabled()); this.updateStatus(); }} />
                    </Col>
                </Row>
                <Row className="text-left">
                    <Col>
                        <Form.Switch id="imgmem" label="Enable Memory Cache" checked={this.state.status.getMemcacheEnabled()}
                            onChange={() => { this.state.status.setMemcacheEnabled(!this.state.status.getMemcacheEnabled()); this.updateStatus(); }} />
                    </Col>
                </Row>
                <Row className="text-left">
                    <Col>
                        <Form.Switch id="imgdisk" label="Enable Disk Cache" checked={this.state.status.getDiskcacheEnabled()}
                            onChange={() => { this.state.status.setDiskcacheEnabled(!this.state.status.getDiskcacheEnabled()); this.updateStatus(); }} />
                    </Col>
                </Row>
                <Row className="text-left">
                    <Col>
                        <Button variant="primary" onClick={() => { this.doJump(); }}>Jump</Button>
                    </Col>
                </Row>
            </Container >
        )
    }
}

export default ImageBoard;