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
import ncaamlogo from './ncaam.png'

var BACKEND = "http://" + window.location.host

class Sport extends React.Component {
    constructor(props) {
        super(props);
        this.state = { "disablerChecked": false };
    }
    callmatrix(path) {
        console.log(`Calling matrix API nhl/${path}`)
        fetch(`${BACKEND}/api/${this.props.sport}/${path}`, {
            method: "GET",
            mode: "cors",
        });
    }

    async componentDidMount() {
        const resp = await fetch(`${BACKEND}/api/${this.props.sport}/status`,
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

    logosrc() {
        if (this.props.sport == "nhl") {
            return nhllogo
        } else if (this.props.sport == "ncaam") {
            return ncaamlogo
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
                        <Form.Switch id="enabler" label="Enable/Disable" checked={this.state.disablerChecked} onChange={this.handleSwitch} />
                    </Col>
                </Row>

                <Row className="text-center"><Col><Button onClick={() => this.callmatrix("hidefavoritescore")}>Hide Favorite Scores</Button></Col></Row>
                <Row className="text-center"><Col><Button onClick={() => this.callmatrix("showfavoritescore")}>Show Favorite Scores</Button></Col></Row>
                <Row className="text-center"><Col><Button onClick={() => this.callmatrix("favoritesticky")}>Sticky Favorite Team</Button></Col></Row>
                <Row className="text-center"><Col><Button onClick={() => this.callmatrix("favoriteunstick")}>Unstick Favorite Team</Button></Col></Row>
            </Container>
        )
    }
}

export default Sport;