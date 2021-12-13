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
import ncaamlogo from './ncaam.png';
import ncaaflogo from './ncaaf.png';
import nbalogo from './nba.png';
import nfllogo from './nfl.png';
import mlslogo from './mls.png';
import epllogo from './epl.png'
import { MatrixPostRet, JSONToStatus, JumpToBoard } from './util';
import { SetStatusReq, Status } from './sportboard/sportboard_pb';
import * as basicboard_pb from './basicboard/basicboard_pb';


function jsonToStatus(jsonDat) {
    var d = JSON.parse(jsonDat);
    var dat = d.status;
    var status = new Status();
    status.setEnabled(dat.enabled);
    status.setScrollEnabled(dat.scroll_enabled);
    status.setFavoriteHidden(dat.favorite_hidden);
    status.setFavoriteSticky(dat.favorite_sticky);
    status.setTightScrollEnabled(dat.tight_scroll_enabled);
    status.setRecordRankEnabled(dat.record_rank_enabled);
    status.setOddsEnabled(dat.odds_enabled);
    status.setUseGradient(dat.use_gradient);

    return status;
}

class Sport extends React.Component {
    constructor(props) {
        super(props);
        var status = new Status();
        this.state = {
            "status": status,
            "stats": new basicboard_pb.Status(),
            "headlines": new basicboard_pb.Status(),
            "has_stats": false,
        };
        if (this.props.sport === "nhl") {
            console.log("Sport created ", this.props.sport, this.state.status)
        }
    }
    async componentDidMount() {
        await this.getStatus()
        if (this.props.sport === "nhl") {
            console.log("Sport Updated " + this.props.sport + " " + this.state.enabled)
        }
    }
    getStatus = async () => {
        await MatrixPostRet(this.props.sport + "/sport.v1.Sport/GetStatus", '{}').then((resp) => {
            if (resp.ok) {
                return resp.text()
            }
            throw resp
        }).then((data) => {
            if (this.props.sport === "nhl") {
                console.log("Got MatrixPostRet", data);
            }
            var dat = jsonToStatus(data);
            this.setState({
                "status": dat,
            })
        });

        await MatrixPostRet("stat/" + this.props.sport + "/board.v1.BasicBoard/GetStatus", '{}').then((resp) => {
            if (resp.ok) {
                return resp.text();
            }
            throw resp
        }).then((data) => {
            if (this.props.sport === "nhl") {
                console.log("Got MatrixPost from stat GetStatus", data);
            }
            try {
                var dat = JSONToStatus(data);
                this.setState({
                    "stats": dat,
                    "has_stats": true,
                })
            } catch (e) {
                this.setState({
                    "has_stats": false,
                });
            }
        }).catch(error => {
            this.setState({
                "has_stats": false,
            });
        });

        await MatrixPostRet("headlines/" + this.props.sport + "/board.v1.BasicBoard/GetStatus", '{}').then((resp) => {
            if (resp.ok) {
                return resp.text();
            }
            throw resp;
        }).then((data) => {
            try {
                var dat = JSONToStatus(data);
                this.setState({
                    "headlines": dat,
                    "has_headlines": true,
                })
            } catch (e) {
                this.setState({
                    "has_headlines": false,
                });
            }
        }).catch(error => {
            this.setState({
                "has_headlines": false,
            });
        });
    }

    updateStatus = async () => {
        var req = new SetStatusReq();
        req.setStatus(this.state.status);
        await MatrixPostRet(this.props.sport + "/sport.v1.Sport/SetStatus", JSON.stringify(req.toObject()));

        var sreq = new basicboard_pb.SetStatusReq();
        sreq.setStatus(this.state.stats);
        await MatrixPostRet("stat/" + this.props.sport + "/board.v1.BasicBoard/SetStatus", JSON.stringify(sreq.toObject()));

        var hreq = new basicboard_pb.SetStatusReq();
        hreq.setStatus(this.state.headlines);
        await MatrixPostRet("headlines/" + this.props.sport + "/board.v1.BasicBoard/SetStatus", JSON.stringify(sreq.toObject()));
        await this.getStatus();
    }

    logosrc() {
        if (this.props.sport === "nhl") {
            return nhllogo
        } else if (this.props.sport === "ncaam") {
            return ncaamlogo
        } else if (this.props.sport === "nhl") {
            return nhllogo
        } else if (this.props.sport === "nba") {
            return nbalogo
        } else if (this.props.sport === "nfl") {
            return nfllogo
        } else if (this.props.sport === "mls") {
            return mlslogo
        } else if (this.props.sport === "epl") {
            return epllogo
        } else if (this.props.sport === "ncaaf") {
            return ncaaflogo
        } else {
            return mlblogo
        }
    }
    render() {
        return (
            <Container fluid>
                <Row className="text-center">
                    <Col>
                        <Image src={this.logosrc()} style={{ height: '100px', width: 'auto' }} onClick={() => { JumpToBoard(this.props.sport); this.updateStatus(); this.props.doSync(); }} fluid />
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
                        <Form.Switch id={this.props.sport + "tightscroller"} label="Back-to-back Scroll Mode" checked={this.state.status.getTightScrollEnabled()}
                            onChange={() => { this.state.status.setTightScrollEnabled(!this.state.status.getTightScrollEnabled()); this.updateStatus(); }} />
                    </Col>
                </Row>
                <Row className="text-left">
                    <Col>
                        <Form.Switch id={this.props.sport + "stats"} label="Stats" checked={this.state.stats.getEnabled()} disabled={!this.state.has_stats}
                            onChange={() => { this.state.stats.setEnabled(!this.state.stats.getEnabled()); this.updateStatus(); }} />
                    </Col>
                </Row>
                <Row className="text-left">
                    <Col>
                        <Form.Switch id={this.props.sport + "headlines"} label="News Headlines" checked={this.state.headlines.getEnabled()} disabled={!this.state.has_headlines}
                            onChange={() => { this.state.headlines.setEnabled(!this.state.headlines.getEnabled()); this.updateStatus(); }} />
                    </Col>
                </Row>
                <Row className="text-left">
                    <Col>
                        <Form.Switch id={this.props.sport + "statscroll"} label="Stats Scroll Mode" checked={this.state.stats.getScrollEnabled()} disabled={!this.state.has_stats}
                            onChange={() => { this.state.stats.setScrollEnabled(!this.state.stats.getScrollEnabled()); this.updateStatus(); }} />
                    </Col>
                </Row>
                <Row className="text-left">
                    <Col>
                        <Form.Switch id={this.props.sport + "favscore"} label="Hide Favorite Scores" checked={this.state.status.getFavoriteHidden()}
                            onChange={() => { this.state.status.setFavoriteHidden(!this.state.status.getFavoriteHidden()); this.updateStatus(); }} />
                    </Col>
                </Row>
                <Row className="text-left">
                    <Col>
                        <Form.Switch id={this.props.sport + "odds"} label="Show Odds" checked={this.state.status.getOddsEnabled()}
                            onChange={() => { this.state.status.setOddsEnabled(!this.state.status.getOddsEnabled()); this.updateStatus(); }} />
                    </Col>
                </Row>
                <Row className="text-left">
                    <Col>
                        <Form.Switch id={this.props.sport + "record"} label="Record + Rank" checked={this.state.status.getRecordRankEnabled()}
                            onChange={() => { this.state.status.setRecordRankEnabled(!this.state.status.getRecordRankEnabled()); this.updateStatus(); }} />
                    </Col>
                </Row>
                <Row className="text-left">
                    <Col>
                        <Form.Switch id={this.props.sport + "favstick"} label="Stick Favorite Live Games" checked={this.state.status.getFavoriteSticky()}
                            onChange={() => { this.state.status.setFavoriteSticky(!this.state.status.getFavoriteSticky()); this.updateStatus(); }} />
                    </Col>
                </Row>
                <Row className="text-left">
                    <Col>
                        <Form.Switch id={this.props.sport + "gradient"} label="Logo Gradient" checked={this.state.status.getUseGradient()}
                            onChange={() => { this.state.status.setUseGradient(!this.state.status.getUseGradient()); this.updateStatus(); }} />
                    </Col>
                </Row>
                <Row className="text-left">
                    <Col>
                        <Button variant="primary" onClick={() => { JumpToBoard(this.props.sport); this.updateStatus(); this.props.doSync(); }}>Jump</Button>
                    </Col>
                </Row>
            </Container>
        )
    }
}

export default Sport;