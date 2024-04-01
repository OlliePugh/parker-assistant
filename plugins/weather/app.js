import express from "express";
import "dotenv/config";

const app = express();
const port = 8080;

app.use(express.json());

app.post("/weather.get", async (req, res) => {
  const location = req.body.locationName ?? "Loughborough";
  const url = new URL("https://api.openweathermap.org/data/2.5/weather");
  url.searchParams.append("q", location);
  url.searchParams.append("APPID", process.env.OPEN_WEATHER_API_KEY);
  url.searchParams.append("units", "metric");

  const result = await fetch(url.href);
  const parsed = await result.json();

  res.send(parseResponse(parsed));
});

const parseResponse = (response) => {
  const { main, weather } = response;
  return {
    temperature: main.temp + "C",
    feelsLike: main.feels_like + "C",
    humidity: main.humidity + "%",
    weather: weather[0].description,
    wind: response.wind.speed + "m/s",
  };
};

app.listen(port, () => {
  console.log(`Weather pluginr running on http://localhost:${port}`);
});
