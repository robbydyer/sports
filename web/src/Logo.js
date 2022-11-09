import nhllogo from './nhllogo.jpeg';
import mlblogo from './mlb.png';
import ncaamlogo from './ncaam.png';
import ncaaflogo from './ncaaf.png';
import nbalogo from './nba.png';
import nfllogo from './nfl.png';
import mlslogo from './mls.png';
import epllogo from './epl.png'
import dfllogo from './dfl.png';
import dfblogo from './dfb.png';
import clock from './clock.png';
import stocks from './stock.png';
import pga from './pga.png';
import sys from './server.png';
import cal from './cal.png';
import weather from './weather.png';
import imgimg from './image.png';
import f1logo from './f1.png';
import irllogo from './irl.png';
import uefa from './uefa.png';
import fifa from './fifa.png'

export function LogoSrc(sport) {
    if (sport === "nhl") {
        return nhllogo
    } else if (sport === "ncaam") {
        return ncaamlogo
    } else if (sport === "nhl") {
        return nhllogo
    } else if (sport === "mlb") {
        return mlblogo
    } else if (sport === "nba") {
        return nbalogo
    } else if (sport === "nfl") {
        return nfllogo
    } else if (sport === "mls") {
        return mlslogo
    } else if (sport === "epl") {
        return epllogo
    } else if (sport === "ncaaf") {
        return ncaaflogo
    } else if (sport === "dfl") {
        return dfllogo
    } else if (sport === "dfb") {
        return dfblogo
    } else if (sport === "pga") {
        return pga
    } else if (sport === "clock") {
        return clock
    } else if (sport === "stocks") {
        return stocks
    } else if (sport === "sys") {
        return sys
    } else if (sport === "gcal") {
        return cal
    } else if (sport === "weather") {
        return weather
    } else if (sport === "img") {
        return imgimg
    } else if (sport === "f1") {
        return f1logo
    } else if (sport === "irl") {
        return irllogo
    } else if (sport === "uefa") {
        return uefa
    } else if (sport === "fifa") {
        return fifa
    }
}