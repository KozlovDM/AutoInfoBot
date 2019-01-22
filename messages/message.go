package messages

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gopkg.in/telegram-bot-api.v4"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type Config struct {
	TelegramBotToken string
	CaptchaApiKey    string
}

type Osago struct {
	InsCompanyName   string `json:"insCompanyName"`
	PolicyBsoNumber  string `json:"policyBsoNumber"`
	PolicyBsoSerial  string `json:"policyBsoSerial"`
	PolicyIsRestrict string `json:"policyIsRestrict"`
	PolicyUnqId      string `json:"policyUnqId"`
}

type Vin struct {
	BodyNumber    string `json:"bodyNumber"`
	ChassisNumber string `json:"chassisNumber"`
	LicensePlate  string `json:"licensePlate"`
	InsurerName   string `json:"insurerName"`
	PolicyStatus  string `json:"policyStatus"`
	Vin           string `json:"vin"`
}

type Owner struct {
	LastOperation    string `json:"lastOperation"`
	SimplePersonType string `json:"simplePersonType"`
	From             string `json:"from"`
	To               string `json:"to"`
}

type AutoInfo struct {
	EngineVolume string `json:"engineVolume"`
	Color        string `json:"color"`
	BodyNumber   string `json:"bodyNumber"`
	Year         string `json:"year"`
	EngineNumber string `json:"engineNumber"`
	Vin          string `json:"vin"`
	Model        string `json:"model"`
	Category     string `json:"category"`
	Type         string `json:"type"`
	PowerHp      string `json:"powerHp"`
	PowerKwt     string `json:"powerKwt"`
}

func Send(configuration Config) {
	bot, err := tgbotapi.NewBotAPI(configuration.TelegramBotToken)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = false

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		log.Panic(err)
	}

	for update := range updates {
		vin, err := getVIN(update.Message.Text, configuration.CaptchaApiKey)
		if err != nil {
			vin, err = getVIN(update.Message.Text, configuration.CaptchaApiKey)
			if err != nil {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "По данному номеру ничего не найденно")
				_, err = bot.Send(msg)
				if err != nil {
					log.Panic(err)
				}
			}
		}
		owners, auto, err := getAutoInfo(vin, configuration.CaptchaApiKey)
		if err != nil {
			owners, auto, err = getAutoInfo(vin, configuration.CaptchaApiKey)
			if err != nil {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "По данному номеру ничего не найденно")
				_, err = bot.Send(msg)
				if err != nil {
					log.Panic(err)
				}
			}
		}
		nameFile, err := getResultFile(update.Message.Text, owners, auto)
		if err != nil {
			nameFile, err = getResultFile(update.Message.Text, owners, auto)
			if err != nil {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "По данному номеру ничего не найденно")
				_, err = bot.Send(msg)
				if err != nil {
					log.Panic(err)
				}
			}
		}
		upload := tgbotapi.NewDocumentUpload(update.Message.Chat.ID, nameFile)
		_, err = bot.Send(upload)
		if err != nil {
			log.Panic(err)
		}
		err = os.Remove(nameFile + ".txt")
		if err != nil {
			log.Println(err)
		}
	}
}

func getVIN(autoNumber string, captchaApiKey string) (res string, err error) {
	osago, err := getOsago(autoNumber, captchaApiKey)
	if err != nil {
		return
	}
	url := "https://dkbm-web.autoins.ru/dkbm-web-1.0/osagovehicle.htm"
	captcha, err := getSolutionCaptchaV2(captchaApiKey)
	if err != nil {
		return
	}
	body := "serialOsago=" + osago.PolicyBsoSerial + "&numberOsago=" + osago.PolicyBsoNumber + "&dateRequest=" + time.Now().Format("01.02.2006") + "&captcha=" + captcha
	fmt.Println(body)
	resp, err := postResRequest(url, body)
	if err != nil {
		return
	}
	var resArray Vin
	err = json.Unmarshal(resp, &resArray)
	return resArray.Vin, err
}

func getOsago(autoNumber string, captchaApiKey string) (res Osago, err error) {
	url := "https://dkbm-web.autoins.ru/dkbm-web-1.0/policy.htm"
	captcha, err := getSolutionCaptchaV2(captchaApiKey)
	if err != nil {
		return
	}
	body := "vin=&lp=" + autoNumber + "&date=" + time.Now().Format("02.01.2006") + "&bodyNumber=&chassisNumber=&captcha=" + captcha
	fmt.Println(body)
	bodyRes, err := postResRequest(url, body)
	if err != nil {
		return
	}
	osago := bytes.Split(bodyRes, []byte("["))
	osago = bytes.Split([]byte(osago[1]), []byte("]"))
	err = json.Unmarshal([]byte(osago[0]), &res)
	return
}

func getAutoInfo(vin string, captchaApiKey string) (ownersRes []Owner, auto AutoInfo, err error) {
	url := "https://xn--b1afk4ade.xn--90adear.xn--p1ai/proxy/check/auto/history"
	captcha, err := getSolutionCaptchaV3(captchaApiKey, "check_auto_history")
	if err != nil {
		return
	}
	body := "vin=" + vin + "&captchaWord=&checkType=history&reCaptchaToken=" + captcha
	fmt.Println(body)
	bodyRes, err := postResRequest(url, body)
	if err != nil {
		return
	}
	split := bytes.Split(bodyRes, []byte("["))
	split = bytes.Split(split[1], []byte("]"))
	split[0] = []byte(strings.Replace(string(split[0]), "},{", "}@{", -1))
	owners := bytes.Split(split[0], []byte("@"))
	var owner Owner
	count := bytes.Count(split[0], []byte("@"))
	ownersRes = make([]Owner, count+1)
	for i := 0; i <= count; i++ {
		err = json.Unmarshal(owners[i], &owner)
		if err != nil {
			return
		}
		ownersRes[i] = owner
	}

	split[1] = []byte(strings.Replace(string(split[1]), ":{", "@{", 2))
	split[1] = []byte(strings.Replace(string(split[1]), "},", "@", 3))
	vehicle := bytes.Split(split[1], []byte("@"))
	err = json.Unmarshal(vehicle[4], &auto)
	return
}

func postResRequest(url string, body string) (res []byte, err error) {
	req, err := http.DefaultClient.Post(url, "application/x-www-form-urlencoded", strings.NewReader(body))
	if err != nil {
		return
	}
	res, err = ioutil.ReadAll(req.Body)
	if err != nil {
		return
	}
	fmt.Println(string(res))
	err = req.Body.Close()
	return
}

func getSolutionCaptchaV2(captchaApiKey string) (solution string, err error) {
	url := "http://rucaptcha.com/in.php?key=" + captchaApiKey + "&method=userrecaptcha&googlekey=6Lf2uycUAAAAALo3u8D10FqNuSpUvUXlfP7BzHOk&pageurl=https://dkbm-web.autoins.ru/dkbm-web-1.0/policy.htm"
	bodyRes, err := getResRequest(url)
	if err != nil {
		return
	}
	split := bytes.Split(bodyRes, []byte("|"))
	time.Sleep(15 * time.Second)
	url = "http://rucaptcha.com/res.php?key=" + captchaApiKey + "&action=get&id=" + string(split[1])

	for ; ; {
		res, err := getResRequest(url)
		if err != nil {
			return "", err
		}
		if strings.EqualFold(string(res), "CAPCHA_NOT_READY") {
			time.Sleep(5 * time.Second)
		} else {
			split := bytes.Split(res, []byte("|"))
			return string(split[1]), nil
		}
	}
}

func getSolutionCaptchaV3(captchaApiKey string, action string) (solution string, err error) {
	url := "http://rucaptcha.com/in.php?key=" + captchaApiKey + "&method=userrecaptcha&version=v3&action=" + action + "&min_score=0.3&googlekey=6Lc66nwUAAAAANZvAnT-OK4f4D_xkdzw5MLtAYFL&pageurl=https://гибдд.рф/check/auto/"
	bodyRes, err := getResRequest(url)
	if err != nil {
		return
	}
	split := bytes.Split(bodyRes, []byte("|"))
	time.Sleep(15 * time.Second)
	url = "http://rucaptcha.com/res.php?key=" + captchaApiKey + "&action=get&id=" + string(split[1])

	for ; ; {
		res, err := getResRequest(url)
		if err != nil {
			return "", err
		}
		if strings.EqualFold(string(res), "CAPCHA_NOT_READY") {
			time.Sleep(5 * time.Second)
		} else {
			split := bytes.Split(res, []byte("|"))
			return string(split[1]), nil
		}
	}
}

func getResRequest(url string) (res []byte, err error) {
	req, err := http.DefaultClient.Get(url)
	if err != nil {
		return
	}
	res, err = ioutil.ReadAll(req.Body)
	if err != nil {
		return
	}
	err = req.Body.Close()
	return
}

func getResultFile(autoNumber string, owners []Owner, auto AutoInfo) (fileName string, err error) {
	fileName = autoNumber + ".txt"
	file, err := os.Create(fileName)
	if err != nil {
		return
	}
	_, err = file.WriteString("Марка, модель: " + auto.Model + "\n" +
		"Год выпуска: " + auto.Year + "\n" +
		"VIN: " + auto.Vin + "\n" +
		"Кузов: " + auto.BodyNumber + "\n" +
		"Шасси: " + auto.Model + "\n" +
		"Цвет: " + auto.Color + "\n" +
		"Рабочий объем (см³): " + auto.EngineVolume + "\n" +
		"Мощность (кВт/л.с.): " + auto.PowerKwt + "/" + auto.PowerHp + "\n" +
		"Категория: " + auto.Category + "\n" +
		"Тип: " + auto.Type + "\n" +
		"Периоды владения транспортным средством\n")

	if err != nil {
		if err := os.Remove(fileName); err != nil {
			log.Println(err)
		}
		return
	}

	for i := 0; i < len(owners); i++ {
		_, err = file.WriteString("C" + owners[i].From + "по" + owners[i].To + ":" + owners[i].SimplePersonType + "\n")
		if err != nil {
			if err := os.Remove(fileName); err != nil {
				log.Println(err)
			}
			return
		}
	}
	return
}
