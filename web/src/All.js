import React from 'react';
import 'bootstrap/dist/css/bootstrap.min.css';
import Container from 'react-bootstrap/Container';
import Row from 'react-bootstrap/Row';
import Col from 'react-bootstrap/Col';
import Card from 'react-bootstrap/Card';
import Home from './Home.js';
import Sport from './Sport.js';
import ImageBoard from './ImageBoard.js';
import Clock from './Clock.js';
import Sys from './Sys.js';
import Pga from './Pga.js';
import Stocks from './Stocks.js';
import Weather from './Weather.js';

const styles = {
    row: {
        marginTop: 10
    },
    col: {
        paddingTop: '20px'
    }
}

const card_border = "18rem"


class All extends React.Component {
    constructor(props) {
        super(props);
        this.state = {
            "sync": Date.now(),
        }
    }


    doSync = () => {
        console.log("All page updating sync time")
        this.setState({ "sync": Date.now() })
    }

    render() {
        var sports = ["ncaaf", "nhl", "mlb", "ncaam", "nfl", "nba", "mls", "epl"].map((sport) =>
            <Col lg="auto" style={styles.col}>
                <Card style={{ width: { card_border } }}>
                    <Sport sport={sport} id={sport} key={sport + this.state.sync} doSync={this.doSync} />
                </Card>
            </Col>
        );
        return (
            <Container fluid="xl">
                <Row className="justify-content-md-space-between" sm={1} lg={2} xl={3} style={styles.row}>
                    <Col lg="auto" style={styles.col}>
                        <Card style={{ width: { card_border } }}>
                            <Home doSync={this.doSync} key={"home" + this.state.sync} />
                        </Card>
                    </Col>
                    {sports}
                    <Col lg="auto" style={styles.col}>
                        <Card style={{ width: { card_border } }}>
                            <Pga doSync={this.doSync} key={"pga" + this.state.sync} />
                        </Card>
                    </Col>
                    <Col lg="auto" style={styles.col}>
                        <Card style={{ width: { card_border } }}>
                            <Weather id="weatherboard" doSync={this.doSync} key={"weather" + this.state.sync} />
                        </Card>
                    </Col>
                    <Col lg="auto" style={styles.col}>
                        <Card style={{ width: { card_border } }}>
                            <ImageBoard id="imgboard" doSync={this.doSync} key={"img" + this.state.sync} />
                        </Card>
                    </Col>
                    <Col lg="auto" style={styles.col}>
                        <Card style={{ width: { card_border } }}>
                            <Stocks id="stocks" doSync={this.doSync} key={"stocks" + this.state.sync} />
                        </Card>
                    </Col>
                    <Col lg="auto" style={styles.col}>
                        <Card style={{ width: { card_border } }}>
                            <Clock id="clock" doSync={this.doSync} key={"clock" + this.state.sync} />
                        </Card>
                    </Col>
                    <Col lg="auto" style={styles.col}>
                        <Card style={{ width: { card_border } }}>
                            <Sys id="sys" doSync={this.doSync} key={"sys" + this.state.sync} />
                        </Card>
                    </Col>
                </Row>
            </Container>
        )
    }
}

export default All;