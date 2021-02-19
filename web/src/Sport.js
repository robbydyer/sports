import React from 'react';
import 'bootstrap/dist/css/bootstrap.min.css';
import Button from 'react-bootstrap/Button';
import Container from 'react-bootstrap/Container';
import Row from 'react-bootstrap/Row';
import Col from 'react-bootstrap/Col';
import Image from 'react-bootstrap/Image';
import nhllogo from './nhllogo.jpeg';
import mlblogo from './mlb.png';

var BACKEND = "http://" + window.location.host

class Sport extends React.Component {
    constructor(props) {
        super(props);
    }
    callmatrix(path) {
        console.log(`Calling matrix API nhl/${path}`)
        fetch(`${BACKEND}/api/${this.props.sport}/${path}`, {
            method: "GET",
            mode: "cors",
        });
    }
    logosrc() {
        if (this.props.sport == "nhl") {
            return nhllogo
        } else {
            return mlblogo
        }
    }
    render() {
        return (
            <Container fluid>
                <Row className="text-center"><Col><Image src={this.logosrc()} style={{ height: '100px', width: 'auto' }} fluid /></Col></Row>
                <Row className="text-center"><Col><Button onClick={() => this.callmatrix("enable")}>Enable</Button></Col></Row>
                <Row className="text-center"><Col><Button onClick={() => this.callmatrix("disable")}>Disable</Button></Col></Row>
                <Row className="text-center"><Col><Button onClick={() => this.callmatrix("hidefavoritescore")}>Hide Favorite Scores</Button></Col></Row>
                <Row className="text-center"><Col><Button onClick={() => this.callmatrix("showfavoritescore")}>Show Favorite Scores</Button></Col></Row>
                <Row className="text-center"><Col><Button onClick={() => this.callmatrix("favoritesticky")}>Sticky Favorite Team</Button></Col></Row>
                <Row className="text-center"><Col><Button onClick={() => this.callmatrix("favoriteunstick")}>Unstick Favorite Team</Button></Col></Row>
            </Container>
        )
    }
}

export default Sport;