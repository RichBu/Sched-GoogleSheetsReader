package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

var fileFullName string = "https://docs.google.com/spreadsheets/d/15m0IGXTTWvHp71oFVmnbmRnbfsA__wYP6j9HrqotfGY/edit?gid=1360672913#gid=1360672913"

var validVenId = []string{
	"1",
	"2",
	"3",
	"4",
	"5",
	"6",
	"7",
	"8",
	"9",
	"10",
	"11",
	"12",
	"13",
	"14",
	"15",
}
var validVenName = []string{
	"Stagg - Turf Field",
	"Stagg - Track",
	"Stagg - Tennis Court",
	"Campus South Field",
	"Ratner - Competition Gym",
	"Ratner - Auxillary Gym",
	"Ratner - Dance Studio",
	"HCFH Upper Level - Track",
	"HCFH Upper Level - Court 1",
	"HCFH Upper Level - Court 2",
	"HCFH Upper Level - Court 3",
	"HCFH Upper Level - Court 4",
	"HCFH - Martial Arts Room",
	"HCFH - Multi-purpose Room",
	"HCFH Upper Level - Balcony",
}
var validVenTab = []string{
	"Stagg Turf Field, Track & Tenni",
	"Stagg Turf Field, Track & Tenni",
	"Stagg Turf Field, Track & Tenni",
	"Campus South Athletic Field",
	"Ratner Competition and Auxiliar",
	"Ratner Competition and Auxiliar",
	"Ratner Dance Studio",
	"HCFH Upper Level",
	"HCFH Upper Level",
	"HCFH Upper Level",
	"HCFH Upper Level",
	"HCFH Upper Level",
	"HCFH Multi-Purpose Room & Marti",
	"HCFH Multi-Purpose Room & Marti",
	"HCFH UL Balcony",
}
var validVenCol = []string{
	"Turf",
	"Track",
	"Tennis",
	"South Field",
	"Competition Gymnasium",
	"Auxiliary Gymnasium",
	"Dance Studio",
	"Track",
	"Ct 1",
	"Ct 2",
	"Ct 3",
	"Ct 4",
	"Martial Arts",
	"MP",
	"Balcony",
}

// global variables
var fileID string
var fileWithExport string
var fileShtTabs []string

type TimeEntryType struct {
	TimeStr string `json:"time"`
	IsFree  bool   `json:"isfree"`
}

type DateEntryType struct {
	DateStrOrig      string          `json:"dateorig"`
	DateStrFormatted string          `json:"dateformatted"`
	TimeEntries      []TimeEntryType `json:"timeentries"`
}

type TimeColType struct {
	TabNum      int             `json:"tabnum"`
	TabName     string          `json:"tabname"`
	ColName     string          `json:"colname"`
	DateEntries []DateEntryType `json:"dateentries"`
}

var timeCols []TimeColType
var timeCols2 []TimeColType

func PrintPretty(inObj any) {
	prettyObj, err := json.MarshalIndent(inObj, "", " ")
	if err != nil {
		fmt.Println("Error marshalling:", err)
	}
	fmt.Println(string(prettyObj))
}

func ReadCol(f *excelize.File, sheet string, targetCell string) (*DateEntryType, error) {
	//read a whole column
	//target cell is the the row2 cell at the top
	//stop when the time goes blank

	var lastDate string
	var timeEntries []TimeEntryType
	var dateEntryRec DateEntryType

	//fmt.Println("ReadCol cell = " + targetCell)

	col, row, err := excelize.CellNameToCoordinates(targetCell)
	if err != nil {
		return nil, err
	}

	row = 2
	cellSearch, err := excelize.CoordinatesToCellName(col, row)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	// call the function now
	//is the cell merged
	isMerged, cellMaster, err := FindMasterCell(f, sheet, cellSearch)
	if err != nil {
		fmt.Println("Error FindMasterCell")
		fmt.Println(err)
		return nil, err
	}
	cellDate := cellSearch
	if isMerged {
		//fmt.Print("merged cell = ")
		//fmt.Println(cellMaster)
		cellDate = cellMaster
	}
	//get date value
	cellDateVal, err := f.GetCellValue(sheet, cellDate)
	if err != nil {
		fmt.Println("Error getting date value")
		fmt.Println(err)
		return nil, err
	}

	/*
		fmt.Println("Date =")
		fmt.Println(cellDateVal)
	*/
	dateStrFormatted, err2 := createDateStr(cellDateVal)
	if err2 != nil {
		fmt.Println("Error converting date")
		fmt.Println(cellDateVal)
		return nil, err2
	}
	lastDate = *dateStrFormatted

	/*
		fmt.Println("-- Cell Date --")
		fmt.Println(cellDate)
		fmt.Println(cellDateVal)
	*/

	//find the time entries
	continTimLp := true
	cellTim_p1 := 'A'
	cellTim_p2_val := 4
	row = 4
	for continTimLp {
		cellTim_p2 := strings.TrimLeft(strconv.Itoa(int(cellTim_p2_val)), "")
		cellSearch := string(cellTim_p1) + string(cellTim_p2)
		//is the cell merged
		isMerged, cellMaster, err := FindMasterCell(f, sheet, cellSearch)
		if err != nil {
			fmt.Println("Error FindMasterCell")
			fmt.Println(err)
			return nil, err
		}

		cellLookup := cellSearch
		if isMerged {
			cellLookup = cellMaster
		}

		//check if the time value not blank
		cellValue, err := f.GetCellValue(sheet, cellLookup) //time value
		if err != nil {
			fmt.Println("Error getting time cell value")
			fmt.Println(err)
			return nil, err
		}
		timeValString := strings.TrimLeft(cellValue, " ")
		timeValString = strings.TrimRight(timeValString, " ")
		timeValString = strings.ReplaceAll(timeValString, " ", "")
		//fmt.Println(cellLookup + " = " + timeValString)
		if (strings.TrimRight(timeValString, " ")) == "" {
			continTimLp = false
		} else {
			//not time cell blank so there are time slots

			//check if the time slot is free
			cellAct, err4 := excelize.CoordinatesToCellName(col, row)
			if err4 != nil {
				fmt.Println("Error in coord to cell name for act")
				fmt.Println(err4)
				return nil, err4
			}

			//is the cell merged
			isMerged, cellMaster, err := FindMasterCell(f, sheet, cellAct)
			if err != nil {
				fmt.Println("Error FindMasterCell for cellAct")
				fmt.Println(err)
				return nil, err
			}

			cellLookup := cellAct
			if isMerged {
				cellLookup = cellMaster
			}

			cellActVal, err := f.GetCellValue(sheet, cellLookup) //actual time value
			if err != nil {
				fmt.Println("Error getting cell value for activity")
				fmt.Println(err)
				return nil, err
			}

			cellIsBlank := false
			if strings.TrimLeft(strings.TrimRight(cellActVal, " "), " ") == "" {
				cellIsBlank = true
			}

			/*
				fmt.Printf("-- cell: %s value: |%s| IsBlank: %v\n", cellLookup, cellActVal, cellIsBlank)
			*/

			// check the style
			styleID, err := f.GetCellStyle(sheet, cellLookup)
			if err != nil {
				fmt.Println("Error getting cell style ID for activity")
				return nil, err
			}
			// Get style definition (available in excelize v2.8.0+)
			style, err := f.GetStyle(styleID)
			if err != nil || style == nil {
				fmt.Println("Error getting cell style for activity")
				return nil, err
			}
			// Check fill color
			cellFillColor := ""
			cellIsFreeColor := true
			if style.Fill.Type != "" {
				colors := style.Fill.Color
				cellFillColor = fmt.Sprintf("%v", colors)
				if cellFillColor != "[FFFFFF]" {
					cellIsFreeColor = false
				}
				//fmt.Printf("%s on %s color: %s is free: %v\n", cellLookup, sheet, cellFillColor, cellIsFreeColor)
			}

			cellIsFree := false
			if cellIsBlank && cellIsFreeColor {
				cellIsFree = true
				/*
					fmt.Println("-- cell is free")
				*/
			}

			timeEntryRec := TimeEntryType{
				TimeStr: timeValString,
				IsFree:  cellIsFree,
			}
			timeEntries = append(timeEntries, timeEntryRec)
		}

		cellTim_p2_val++
		row++
	} // all the rows are done

	// the rows are done, store the record
	dateEntryRec = DateEntryType{
		DateStrOrig:      cellDateVal,
		DateStrFormatted: lastDate,
		TimeEntries:      timeEntries,
	}

	/*
		fmt.Println("---- dataEntryRec object ----")
		PrintPretty((dateEntryRec))
		waitKeyPress()
	*/

	return &dateEntryRec, nil
}

func StrPrnDateEntryRec(dataRecIn DateEntryType) []string {
	// string print data entry rec
	var outVal []string

	for _, timeRec := range dataRecIn.TimeEntries {
		width := 10
		leftPad := (width - 1) / 2
		ch := ' '
		if timeRec.IsFree {
			ch = ' '
		} else {
			ch = 'X'
		}
		outStr := fmt.Sprintf("%*s%c%*s ", leftPad, "", ch, width-1-leftPad, "")
		outVal = append(outVal, outStr)
	}
	return outVal
}

func StrPrnDateTitles(dataRecIn DateEntryType) []string {
	// string print data entry rec left hand side titles
	var outVal []string

	for _, timeRec := range dataRecIn.TimeEntries {
		outStr := fmt.Sprintf("%13s", timeRec.TimeStr)
		outVal = append(outVal, outStr)
	}
	return outVal
}

func StrPrnDateHeading1() string {
	// string for the heading of the columns
	var outVal string

	outVal = fmt.Sprintf("%2s-%13s ", "#", "Time    ")
	/*
		for _, timeRec := range dataRecIn.TimeEntries {
			outStr := fmt.Sprintf("%15s", timeRec.TimeStr)
		}
	*/
	return outVal
}
func StrPrnDateHeading2(dateStr string) string {
	// string for the heading of the columns
	var outVal string

	outVal = fmt.Sprintf(" %10s", dateStr)
	/*
		for _, timeRec := range dataRecIn.TimeEntries {
			outStr := fmt.Sprintf("%15s", timeRec.TimeStr)
		}
	*/
	return outVal
}

func FindMasterCell(f *excelize.File, sheet, targetCell string) (bool, string, error) {
	// Convert target cell name to [col, row] coordinates
	targetCol, targetRow, err := excelize.CellNameToCoordinates(targetCell)
	if err != nil {
		return false, "", err
	}

	mergeCells, err := f.GetMergeCells(sheet)
	if err != nil {
		return false, "", err
	}

	for _, m := range mergeCells {
		startAxis := m.GetStartAxis()
		endAxis := m.GetEndAxis()

		x1, y1, err1 := excelize.CellNameToCoordinates(startAxis)
		x2, y2, err2 := excelize.CellNameToCoordinates(endAxis)
		if err1 != nil || err2 != nil {
			continue
		}

		// Check if target coordinates fall within the rectangle [x1, y1] to [x2, y2]
		if targetCol >= x1 && targetCol <= x2 && targetRow >= y1 && targetRow <= y2 {
			return true, startAxis, nil
		}
	}

	return false, "", nil
}

func createDateStr(dateIn string) (*string, error) {
	// split the date coming from the XLS into a correct string
	outVal := ""

	tempString1 := strings.TrimLeft(dateIn, " ")
	tempString1 = strings.TrimRight(tempString1, " ")
	if tempString1 == "" {
		//fmt.Println("Empty date")
		outVal = ""
		return &outVal, nil
	} else {
		dateParts := strings.Split(tempString1, " ")

		//fmt.Println("date parts:")
		//fmt.Println(dateIn)
		//fmt.Println(tempString1)
		//fmt.Println(dateParts)
		//fmt.Println(dateParts[0])
		//fmt.Println(dateParts[1])

		monthStr := dateParts[0]
		dayStr := dateParts[1]

		t, err := time.Parse("January", monthStr)
		if err != nil {
			fmt.Println("... error on full name ...")
			t2, err12 := time.Parse("Jan", monthStr)
			if err12 != nil {
				fmt.Println("... error on short name also")
				return nil, err
			}
			t = t2
		}

		monthVal := t.Month()
		dayVal, err := strconv.Atoi(dayStr)
		if err != nil {
			fmt.Println("Error converting date string to int")
			fmt.Println(err)
			return nil, err
		}
		yearStr := time.Now().Format("2006")
		dateFullStr := fmt.Sprintf("%02d/%02d/%4s", monthVal, dayVal, yearStr)
		outVal = dateFullStr
		return &outVal, nil
	}
}

func waitKeyPress() {
	fmt.Println("Press Enter to continue...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
	fmt.Println("Continuing !")
}

func checkVenID(venID string) bool {
	outVal := false
	return outVal
}

func readCSV(url string) {
	// read the google sheets as CSV
	// 2. Fetch the public sheet via HTTP GET
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("Failed to fetch data: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Error: server returned status code %d. Make sure the sheet is shared as 'Anyone with the link can view'", resp.StatusCode)
	}

	// 3. Parse the CSV data
	reader := csv.NewReader(resp.Body)
	records, err := reader.ReadAll()
	if err != nil {
		log.Fatalf("Failed to parse CSV data: %v", err)
	}

	// 4. Iterate and print rows
	for rowIndex, row := range records {
		fmt.Printf("Row %d: %v\n", rowIndex+1, row)
	}
}

func readXLSX(url string) {
	// read XLSX

	//find the correct sheet by search by name
	//read sheet
	//read merged cells
	//get count of columns
	//check row 1

	fmt.Println("-----------------------------------")
	fmt.Println("XLSX file")
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error in http Get")
		panic(err)
	}
	defer resp.Body.Close()

	fmt.Println("Status Code = " + strconv.Itoa(resp.StatusCode))
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Status error code")
		panic(fmt.Sprintf("unexpected status code: %d", resp.StatusCode))
	}

	// 4. Convert bytes to string and print

	// Stream the response body directly into Excelize
	f, err := excelize.OpenReader(resp.Body) //orig
	if f == nil {
		fmt.Println("OpenReader is returning nil")
	}
	fmt.Println("Error2 = ")

	defer func() {
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	// Get all sheets (tabs) names
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		panic("no sheets found")
	}

	fmt.Println("Sheets : ")
	for rec, sheet := range sheets {
		sheetCleanUp := strings.ReplaceAll(sheet, "\u0026 ", "& ")
		//sheets[rec] = sheetCleanUp
		fileShtTabs = append(fileShtTabs, sheetCleanUp)
		fmt.Println("  " + strconv.Itoa(rec) + " = " + sheetCleanUp)
	}

	fmt.Println(fileShtTabs)
	waitKeyPress()

	//get all the date records
	//var allDateRecs []DateEntryType
	//try to fill in the data now
	for sheetNum, sheet := range sheets {
		var tempColNames []string //resets for every sheet whereas the timeColRec does not

		cellColName_p1 := 'C'
		cellColName_p2 := '3'
		cellColName := string(cellColName_p1) + string(cellColName_p2)
		cellColNameCol, _, err3 := excelize.CellNameToCoordinates(cellColName)
		if err3 != nil {
			fmt.Println("Converting cell col name")
			fmt.Println(err3)
			return
		}

		// loop thru all the column headings
		// stop when there is a blank
		colLp := true

		for colLp {
			//get the column name
			cellColName, _ := excelize.CoordinatesToCellName(cellColNameCol, 3)
			//fmt.Println("searching : " + cellColName)
			cellValue, err := f.GetCellValue(sheet, cellColName)
			if err != nil {
				fmt.Println("Error getting cell value for cell col name")
				fmt.Printf("|%s|", cellColName)
				fmt.Println(err)
				return
			}
			searchColName := strings.ToLower(strings.TrimRight(cellValue, " "))
			if searchColName == "" {
				//last column on sheet
				colLp = false
				break
			}

			// check if it exists already
			foundCol := false
			for _, tempColName := range tempColNames {
				if strings.ToLower(tempColName) == searchColName {
					foundCol = true
					break
				}
			}

			var dateEntryRec DateEntryType

			//**read a column``
			cellColName, _ = excelize.CoordinatesToCellName(cellColNameCol, 2)
			dateEntryReadIn, _ := ReadCol(f, sheet, cellColName)
			dateEntryRec = *dateEntryReadIn

			if foundCol == false {
				//have not found then create a new record
				// now, store to array to store
				tempColNames = append(tempColNames, searchColName)

				//create new time column for the col name
				// create new record and append
				timeColRecRead := TimeColType{
					TabNum:  sheetNum,
					TabName: sheet,
					ColName: searchColName,
				}
				timeColRecRead.DateEntries = append(timeColRecRead.DateEntries, dateEntryRec)
				timeCols = append(timeCols, timeColRecRead)
			} else {
				//find the time col that matches
				matchRecNum := 0
				for recNum, timeCol := range timeCols {
					if (timeCol.TabName == sheet) && (timeCol.ColName == searchColName) {
						matchRecNum = recNum
						break
					}
				}
				timeCols[matchRecNum].DateEntries = append(timeCols[matchRecNum].DateEntries, dateEntryRec)
			}

			cellColNameCol++

		} //column loop

	} //sheets loop

	/*
		// print out all the timeCols
		waitKeyPress()

		fmt.Println("Tabs:")
		fmt.Println(fileShtTabs)
		fmt.Println("Column names:")
		fmt.Println(timeCols)
		timeColsPretty, err := json.MarshalIndent(timeCols, "", " ")
		if err != nil {
			fmt.Println("Error marshalling:", err)
		}
		fmt.Println(string(timeColsPretty))
	*/

	waitKeyPress()
	fmt.Println("Print column format")
	for _, timeCol := range timeCols {
		// step thru all of the columns
		fmt.Println("Tab num:  " + strconv.Itoa(timeCol.TabNum))
		fmt.Println("Tab name: " + timeCol.TabName)
		fmt.Println("Field:    " + timeCol.ColName)

		// loop thru all the date entries
		printHeadStr := StrPrnDateHeading1()
		for _, dateEntry := range timeCol.DateEntries {
			printHeadStr = printHeadStr + StrPrnDateHeading2(dateEntry.DateStrFormatted)
		}
		fmt.Println(printHeadStr) //heading string at the top
		printTitles := StrPrnDateTitles(timeCol.DateEntries[0])
		printDataLine := ""
		// have left side of line, now need to loop thru the time values (rows)
		for row, _ := range timeCol.DateEntries[0].TimeEntries {
			printDataLine = fmt.Sprintf("%02d-", row) + printTitles[row] + " "
			//loop thru all of the dates
			for _, dateEntry := range timeCol.DateEntries {
				// ToDo: have another one for the desired row only
				printDataRecs := StrPrnDateEntryRec(dateEntry) //it's doing all of them everytime
				printDataLine = printDataLine + printDataRecs[row]
			}
			fmt.Println(printDataLine)
		}
		fmt.Println("-------------------------------------------------------------------------------------------------")
	}
}

/*
// backup of the old function
func readXLSX_backup(url string) {
	// read XLSX

	//find the correct sheet by search by name
	//read sheet
	//read merged cells
	//get count of columns
	//check row 1


	fmt.Println("-----------------------------------")
	fmt.Println("XLSX file")
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error in http Get")
		panic(err)
	}
	defer resp.Body.Close()

	fmt.Println("Status Code = " + strconv.Itoa(resp.StatusCode))
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Status error code")
		panic(fmt.Sprintf("unexpected status code: %d", resp.StatusCode))
	}

	// 4. Convert bytes to string and print

	// Stream the response body directly into Excelize
	f, err := excelize.OpenReader(resp.Body) //orig
	if f == nil {
		fmt.Println("OpenReader is returning nil")
	}
	fmt.Println("Error2 = ")

	defer func() {
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	// Get all sheets (tabs) names
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		panic("no sheets found")
	}

	fmt.Println("Sheets : ")
	for rec, sheet := range sheets {
		sheetCleanUp := strings.ReplaceAll(sheet, "\u0026 ", "& ")
		//sheets[rec] = sheetCleanUp
		fileShtTabs = append(fileShtTabs, sheetCleanUp)
		fmt.Println("  " + strconv.Itoa(rec) + " = " + sheetCleanUp)
	}

	fmt.Println(fileShtTabs)
	waitKeyPress()

	//get all the date records
	var allDateRecs []DateEntryType
	//try to fill in the data now
	for sheetNum, sheet := range sheets {
		var tempColNames []string //resets for every sheet whereas the timeColRec does not

		cellSearch_p1 := 'C'
		cellSearch_p2 := '3'

		// loop thru all the column headings
		// stop when there is a duplicate
		continLp := true
		for continLp {
			cellSearch := string(cellSearch_p1) + string(cellSearch_p2)
			fmt.Println("searching : " + cellSearch)

			cellValue, err := f.GetCellValue(sheet, cellSearch)
			if err != nil {
				fmt.Println("Error getting cell value")
				fmt.Println(err)
				return
			}
			searchColName := strings.ToLower(strings.TrimRight(cellValue, " "))

			// check if it exists already
			foundCol := false
			for _, tempColName := range tempColNames {
				fmt.Println()
				if strings.ToLower(tempColName) == searchColName {
					foundCol = true
					break
				}
			}

			if foundCol == false {
				//have not found then create a new record
				// now, store to array to store
				var timeEntries []TimeEntryType
				sheetCleanUp := strings.ReplaceAll(sheet, "\u0026 ", "& ")

				//get the date and create a date record
				cellDate_p1 := 'C'
				cellDate_p2 := '2'
				cellSearch := string(cellDate_p1) + string(cellDate_p2)

				// call the function now
				//is the cell merged
				isMerged, cellMaster, err := FindMasterCell(f, sheet, cellSearch)
				if err != nil {
					fmt.Println("Error FindMasterCell")
					fmt.Println(err)
					return
				}
				cellDate := cellSearch
				if isMerged {
					fmt.Print("merged cell = ")
					fmt.Println(cellMaster)
					cellDate = cellMaster
				}
				//get date value
				cellDateVal, err := f.GetCellValue(sheet, cellDate)
				if err != nil {
					fmt.Println("Error getting date value")
					fmt.Println(err)
					return
				}

				//find the time entries
				var dateEntryRec DateEntryType
				var dateEntryRead DateEntryType
				continTimLp := true
				cellTim_p1 := "A"
				cellTim_p2_val := 4
				for continTimLp {
					cellTim_p2 := strings.TrimLeft(strconv.Itoa(int(cellTim_p2_val)), "")
					cellSearch := string(cellTim_p1) + string(cellTim_p2)
					//is the cell merged
					isMerged, cellMaster, err := FindMasterCell(f, sheet, cellSearch)
					if err != nil {
						fmt.Println("Error FindMasterCell")
						fmt.Println(err)
						return
					}

					cellLookup := cellSearch
					if isMerged {
						cellLookup = cellMaster
					}
					//check if the time value not blank
					cellValue, err := f.GetCellValue(sheet, cellLookup)
					if err != nil {
						fmt.Println("Error getting cell value")
						fmt.Println(err)
						return
					}
					fmt.Println(cellLookup + " = " + cellValue)
					if (strings.TrimRight(cellValue, " ")) == "" {
						fmt.Println("End search")
						continTimLp = false
					} else {
						//not time cell blank so there are time slots

						//check if the time slot is free
						cellAct_p1 := "C" //second part is the same row as time above
						cellAct := string(cellAct_p1) + string(cellTim_p2)

						//is the cell merged
						isMerged, cellMaster, err := FindMasterCell(f, sheet, cellAct)
						if err != nil {
							fmt.Println("Error FindMasterCell for cellAct")
							fmt.Println(err)
							return
						}

						cellLookup := cellAct
						if isMerged {
							cellLookup = cellMaster
						}

						cellActVal, err := f.GetCellValue(sheet, cellLookup)
						if err != nil {
							fmt.Println("Error getting cell value for activity")
							fmt.Println(err)
							return
						}

						cellIsBlank := false
						if strings.TrimLeft(strings.TrimRight(cellActVal, " "), " ") == "" {
							cellIsBlank = true
						}

						fmt.Printf("-- cell value: |%s| IsBlank: %v\n", cellActVal, cellIsBlank)

						//check the style
						styleID, err := f.GetCellStyle(sheet, cellLookup)
						if err != nil {
							fmt.Println("Error getting cell style ID for activity")
							fmt.Println(err)
						}
						// 2. Get style definition (available in excelize v2.8.0+)
						style, err := f.GetStyle(styleID)
						if err != nil || style == nil {
							fmt.Println("Error getting cell style for activity")
							fmt.Println(err)
						}
						// 3. Check fill color
						cellFillColor := ""
						cellIsFreeColor := true
						if style.Fill.Type != "" {
							colors := style.Fill.Color
							cellFillColor = fmt.Sprintf("%v", colors)
							if cellFillColor != "[FFFFFF]" {
								cellIsFreeColor = false
							}
							fmt.Printf("%s on %s color: %s is free: %v\n", cellLookup, sheet, cellFillColor, cellIsFreeColor)
							//fmt.Printf("  %s has fill color: %v (Pattern: %d) = '%s' \n", cell, colors, style.Fill.Pattern, cellValue)
						}

						cellIsFree := false
						if cellIsBlank && cellIsFreeColor {
							cellIsFree = true
							fmt.Println("-- cell is free")
						}

						timeValString := strings.TrimLeft(cellValue, " ")
						timeValString = strings.TrimRight(timeValString, " ")
						timeValString = strings.ReplaceAll(timeValString, " ", "")
						timeEntryRec := TimeEntryType{
							TimeStr: timeValString,
							IsFree:  cellIsFree,
						}
						timeEntries = append(timeEntries, timeEntryRec)
						dateEntryRec = DateEntryType{
							DateStrOrig: cellDateVal,
							TimeEntries: timeEntries,
						}
						allDateRecs = append(allDateRecs, dateEntryRec)

						//**read a column
						dateEntryReadIn, _ := ReadCol(f, sheet, cellAct_p1+"2")
						dateEntryRead = *dateEntryReadIn
					}

					cellTim_p2_val++
				} //stepping thru all of the rows

				timeColRec := TimeColType{
					TabNum:      sheetNum,
					TabName:     sheetCleanUp,
					ColName:     searchColName,
					DateEntries: dateEntryRec,
				}
				// append a the record
				timeCols = append(timeCols, timeColRec)
				tempColNames = append(tempColNames, searchColName)


					dateOrigStr := dateEntryRead.DateStrOrig
					dateStrFormatted, err2 := createDateStr(dateOrigStr)
					if err2 != nil {
						fmt.Println("Error converting date")
						return
					}
					dateEntryRead.DateStrFormatted = *dateStrFormatted

				timeColRecRead := TimeColType{
					TabNum:      sheetNum,
					TabName:     sheet,
					ColName:     searchColName,
					DateEntries: dateEntryRead,
				}
				// append a the record
				timeCols2 = append(timeCols2, timeColRecRead)

			} else {
				//there was a duplicate so stop
				continLp = false
			}
			cellSearch_p1++
		}

	}


	waitKeyPress()

	fmt.Println("Tabs:")
	fmt.Println(fileShtTabs)
	fmt.Println("Column names:")
	fmt.Println(timeCols)
	timeColsPretty, err := json.MarshalIndent(timeCols, "", " ")
	if err != nil {
		fmt.Println("Error marshalling:", err)
	}
	fmt.Println(string(timeColsPretty))
	waitKeyPress()

	//now iterate thru the sheets
	sheetName := sheets[1]

	// set certain values to blank
	cellCoords := []string{"C6", "D6", "E6", "F6", "AE9", "AP10", "AE10", "AM9", "AN9", "AO9", "AP9", "AQ9", "AQ10", "C4", "D4"}
	cellTextVal := "TEST"
	for _, cellCoord := range cellCoords {
		f.SetCellValue(sheetName, cellCoord, cellTextVal)
	}

	waitKeyPress()
	fmt.Println("Column print")
	dateEntryRec := timeCols2[1].DateEntries
	colTitles := StrPrnDateTitles(dateEntryRec)
	colValues := StrPrnDateEntryRec(dateEntryRec)

	// combine into one string
	var outStrValues []string
	for recNum, colTitleRec := range colTitles {
		tempStr := fmt.Sprintf("%2s-%s=%s", strconv.Itoa(recNum), colTitleRec, colValues[recNum])
		outStrValues = append(outStrValues, tempStr)
	}

	//print headings
	headStr := StrPrnDateHeading(dateEntryRec.DateStrFormatted)
	fmt.Println(headStr)
	// print out values
	for _, outStr := range outStrValues {
		fmt.Println(outStr)
	}

}
*/

func main() {
	fmt.Println("App started.")
	waitKeyPress()
	fmt.Println("Reading an online Google Spreadsheet")
	fmt.Println("")
	fmt.Println("Full file name : " + fileFullName)

	//split the full name into the spreadsheet ID
	splitStart := "/d/"
	splitEnd := "/edit"

	//need to cut the //docs
	_, afterDocs, found := strings.Cut(fileFullName, "//docs")
	if !found {
		fmt.Println("Can not find //docs")
		return
	}

	//cut at start
	_, afterStart, found := strings.Cut(afterDocs, splitStart)
	if !found {
		fmt.Println("Can not find start delimiter")
		return
	}

	//cut the end
	fileIDparse, _, found := strings.Cut(afterStart, splitEnd)
	if !found {
		fmt.Println("ending delimiter not found")
		return
	}
	fileID = fileIDparse
	fmt.Println("File ID        : " + fileID)

	//cut the /edit
	fileExportParse, _, found := strings.Cut(fileFullName, "/edit")
	if !found {
		fmt.Println("ending /edit not found")
		return
	}
	fmt.Println("File w/o edit  : " + fileExportParse)

	fileID = fileExportParse + "/export?format=csv"

	//url := fmt.Sprintf("https://google.com", fileID)
	url := fileID
	fmt.Println("File url       : " + url)

	fmt.Println(" ")
	fmt.Println("CVS File ...")

	// read the CSV export
	//readCSV(url)

	//create the url for XLSX export
	urlXLSX := fileExportParse + "/export?format=xlsx"
	readXLSX(urlXLSX)
}
