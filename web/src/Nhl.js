import React from 'react';
import 'bootstrap/dist/css/bootstrap.min.css';
import Button from 'react-bootstrap/Button';
import Container from 'react-bootstrap/Container';
import Row from 'react-bootstrap/Row';
import Col from 'react-bootstrap/Col';
import Image from 'react-bootstrap/Image';
import nhllogo from './nhllogo.jpeg';

const MATRIX = "http://matrix.local:8080"

class Nhl extends React.Component {
    callmatrix(path) {
        console.log(`Calling matrix API nhl/${path}`)
        fetch(`${MATRIX}/nhl/${path}`, {
            method: "GET",
            mode: "cors",
        });
    }
    render() {
        return (
            <Container fluid>
                <Row className="text-center"><Col><Image src={nhllogo} style={{ height: '100px', width: 'auto' }} fluid /></Col></Row>
                <Row className="text-center"><Col><Button onClick={() => this.callmatrix("enable")}>Enable</Button></Col></Row>
                <Row className="text-center"><Col><Button onClick={() => this.callmatrix("disable")}>Disable</Button></Col></Row>
                <Row className="text-center"><Col><Button onClick={() => this.callmatrix("hidefavoritescore")}>Hide Favorite Scores</Button></Col></Row>
                <Row className="text-center"><Col><Button onClick={() => this.callmatrix("showfavoritescore")}>Show Favorite Scores</Button></Col></Row>
                <Row className="text-center"><Col><Button onClick={() => this.callmatrix("favoritesticky")}>Sticky Favorite Game</Button></Col></Row>
                <Row className="text-center"><Col><Button onClick={() => this.callmatrix("favoriteunstick")}>Unstick Favorite Game</Button></Col></Row>
            </Container>
        )
    }
}

export default Nhl;