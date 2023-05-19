import React from 'react';
import 'bootstrap/dist/css/bootstrap.min.css';
import Container from 'react-bootstrap/Container';
import Row from 'react-bootstrap/Row';
import Col from 'react-bootstrap/Col';
import Image from 'react-bootstrap/Image';

var BACKEND = "http://" + window.location.host
class Board extends React.Component {
    constructor(props) {
        super(props);
        this.state = {
            t: Date.now(),
        }
    }

    componentDidMount() {
        document.body.style.backgroundColor = "black"
        this.interval = setInterval(() => this.setState({ t: Date.now() }), 2000)
    }
    componentWillUnmount() {
        clearInterval(this.interval)
        fetch(`${BACKEND}/api/imgcanvas/disable`, {
            method: "GET",
            mode: "cors",
        });
        document.body.style.backgroundColor = "white"
    }
    render() {
        return (
            <>
                <div
                    style={{
                        backgroundColor: 'black'
                    }}
                />
                <Container fluid>
                    <Row className="text-center"><Col><Image src={`${BACKEND}/api/imgcanvas/board?${this.state.t}`} style={{ height: 'auto', width: 'auto' }} name={this.state.t} fluid /></Col></Row>
                </Container>
            </>
        )
    }
}

export default Board;