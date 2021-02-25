import React from 'react';
import 'bootstrap/dist/css/bootstrap.min.css';
import Button from 'react-bootstrap/Button';
import Container from 'react-bootstrap/Container';
import Row from 'react-bootstrap/Row';
import Col from 'react-bootstrap/Col';
import Image from 'react-bootstrap/Image';
import imgimg from './image.png';
import Form from 'react-bootstrap/Form';

var BACKEND = "http://" + window.location.host

class ImageBoard extends React.Component {
    constructor(props) {
        super(props);
        this.state = { "disablerChecked": false };
    }
    callmatrix(path) {
        console.log(`Calling matrix Image Board /img/${path}`)
        fetch(`${BACKEND}/api/img/${path}`, {
            method: "GET",
            mode: "cors",
        });
    }
    async componentDidMount() {
        const resp = await fetch(`${BACKEND}/api/img/status`,
            {
                method: "GET",
                mode: "cors",
            }
        );

        const status = await resp.text();

        if (resp.ok) {
            if (status === "true") {
                console.log("board is enabled", status)
                this.setState({ "disablerChecked": true })
            } else {
                console.log("board is disabled", status)
                this.setState({ "disablerChecked": false })
            }
        }
    }

    handleSwitch = () => {
        if (!this.state.disablerChecked) {
            console.log("enabling board")
            this.callmatrix("enable")
        } else {
            console.log("disabling board")
            this.callmatrix("disable")
        }
        this.setState({ "disablerChecked": !this.state.disablerChecked })
    }
    render() {
        return (
            <Container fluid>
                <Row className="text-center"><Col><Image src={imgimg} style={{ height: '100px', width: 'auto' }} fluid /></Col></Row>
                <Row className="text-center">
                    <Col>
                        <Form.Switch id="enabler" label="Enable/Disable" checked={this.state.disablerChecked} onChange={this.handleSwitch} />
                    </Col>
                </Row>
                <Row className="text-center"><Col><Button onClick={() => this.callmatrix("enablememcache")}>Enable Memory Cache</Button></Col></Row>
                <Row className="text-center"><Col><Button onClick={() => this.callmatrix("disablememcache")}>Disable Memory Cache</Button></Col></Row>
            </Container>
        )
    }
}

export default ImageBoard;