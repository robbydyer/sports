import React from 'react';
import 'bootstrap/dist/css/bootstrap.min.css';
import Button from 'react-bootstrap/Button';
import Container from 'react-bootstrap/Container';
import Row from 'react-bootstrap/Row';
import Col from 'react-bootstrap/Col';
import Image from 'react-bootstrap/Image';
import conf from './config.json';

class Board extends React.Component {
    constructor(props) {
        super(props);
        this.state = {
            t: Date.now(),
        }
    }

    componentDidMount() {
        this.interval = setInterval(() => this.setState({ t: Date.now() }), 2000)
    }
    componentWillUnmount() {
        clearInterval(this.interval)
    }
    render() {
        return (
            <Container fluid>
                <Row className="text-center"><Col><Image src={`${conf.BACKEND}/api/board?${this.state.t}`} style={{ height: 'auto', width: 'auto' }} name={this.state.t} fluid /></Col></Row>
            </Container>
        )
    }
}

export default Board;