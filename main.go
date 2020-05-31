package main

import (
	"database/sql"
	"log"
	"strconv"
	"net/http"
	"text/template"
	_ "github.com/go-sql-driver/mysql"
	"fmt"
)
//Struct for Invoice items
type InvData struct{
	InvNo	int
	Smen	string
	InvDt	string
	CustNm	string
	InvAmt	float32
	SaleMod	string
}
//Strcut for item details combied with smen allocation 
type ItmDetais struct{
	SmenID  	int
	SmenName	string
	ProdID		int
	ProdDesc	string
	PrdPrce		float32	
	PrdQty		int
	PrdWQty		int
	Extra		float32 // used while picking item details to show particular invoice in Showinvoice page
}
//Struct for Expense
type Expenses struct{
	Dt	string
	Desc	string
	Amt	float32
	Camt	float32
}	
//Struct for Product 
type Product struct{
	Id	int
	Desc	string	
	Qty	int
	Price	float32
}
//Structure for Account Heads
type Acc_head struct{
	Id	int
	Desc	string	
}



//Structure for Showing Salesmen daily morning assignment
type Smen_trans struct{
	Dt	string
	Prod 	string
	Qty 	int
	Smen	string
}

//to handle salesmen values
type Smen struct {
	Id	int
	Desc	string
}

func dbConn() (db *sql.DB){
	dbDriver := "mysql"
	dbUser	:= "niffdbadmin"
	dbPass	:= "nn1234"
	dbName	:= "niffdb"
	db, err := sql.Open(dbDriver,dbUser+":"+dbPass+"@/"+dbName)
	if err != nil {
		panic(err.Error())
	}
	return db
}

var tmpl = template.Must(template.ParseGlob("forms/*"))

func Index( w http.ResponseWriter, r *http.Request){
	tmpl.ExecuteTemplate(w, "Index", nil)	
}

//Handling login,Timebeeing Hardcoded
func Pwd_Verify(w http.ResponseWriter, r *http.Request){
	if r.Method == "POST" {
		uname := r.FormValue("uname")
		pwd := r.FormValue("pwd")
		
		if uname == "niffin" && pwd == "midmac" {
			tmpl.ExecuteTemplate(w, "Home", nil)	
		}else{
			tmpl.ExecuteTemplate(w, "LoginErr", nil)	
		}
	}
}


//Expense
func Expense(w http.ResponseWriter, r *http.Request){
    	db := dbConn()
    	if r.Method == "POST" {
		if "on"==r.FormValue("OptionCheck") {//New Catagory Operation
			new_cat := r.FormValue("new_catagory")
        		insForm, err := db.Prepare("INSERT INTO account_heads(head_desc) VALUES(?)")
        		if err != nil {
           			panic(err.Error())
        		}
			insForm.Exec(new_cat)

		}else{//Adding Expense
       			amount := r.FormValue("Amount")
			date   := r.FormValue("expensedate")
			exp_catagory := r.FormValue("selected_head")
        		insForm, err := db.Prepare("INSERT INTO expense_master(exp_dt,head_id,exp_amt) VALUES (?,?,?)")
        		if err != nil {
           			panic(err.Error())
        		}
			insForm.Exec(date,exp_catagory,amount)
		}

    }
	//Populating account head drop down select tag 
	selDB, err := db.Query("SELECT * FROM account_heads ORDER BY head_id")
	if err !=nil {
		panic(err.Error())
	}
	acchd := Acc_head{}
	acchd_collections := []Acc_head{}

	for selDB.Next() {
		var id int
		var desc string
		err = selDB.Scan(&id, &desc)
		if err != nil {
			panic(err.Error())
		}
	acchd.Id = id
	acchd.Desc = desc
	acchd_collections = append(acchd_collections,acchd)
	}
	
    tmpl.ExecuteTemplate(w, "Expense", acchd_collections)
    defer db.Close()
     

//  http.Redirect(w, r, "/login", 301)
}	


//Salemen Tracking
func Smentracker(w http.ResponseWriter, r *http.Request){
	db := dbConn()
	var strSQL string
       	if r.Method == "POST" {
	rowCount := 1
	for{ // picking data and adding to DB till the row ends

		iCodeInputName := fmt.Sprintf("%s%d","itemCodeRow__",rowCount) 
		iQTYInputName := fmt.Sprintf("%s%d","itemQtyRow__",rowCount) 
		iCode := r.FormValue(iCodeInputName)
		iQty := r.FormValue(iQTYInputName)
		issueDt := r.FormValue("items_issueDt")
		smen_id := r.FormValue("selectSmen")
		if iCode == ""{
			break
		}
		insForm, err := db.Prepare("INSERT INTO smen_trans(smen_id,trans_dt,prod_id,qty) VALUES (?,?,?,?)")
        	if err != nil {
           		panic(err.Error())
        	}
		insForm.Exec(smen_id,issueDt,iCode,iQty)
		
		//Need to update smen_stock table too
		var isProductExists bool
		//First step, need to ensure that details is already in smen_stock, if yes just update otherwise insert
		err= db.QueryRow("SELECT IF(COUNT(*),'true','false') FROM smen_stock WHERE smen_id= ? AND prod_id = ?", smen_id,iCode).Scan(&isProductExists)
		if err != nil {
    			log.Fatal(err)
		} 
		
		if isProductExists {//Updating existing record by just increseing stock 
			strSQL = "UPDATE smen_stock SET assigned_qty = assigned_qty + ? WHERE prod_id = ? AND smen_id = ?"
		}else{
			strSQL = "INSERT INTO smen_stock(assigned_qty,prod_id,smen_id) VALUES(?,?,?)"
		}
		fmt.Println(isProductExists,strSQL)
	                insForm, err = db.Prepare(strSQL)
       		        if err != nil {
                	        panic(err.Error())
                	}
                	insForm.Exec(iQty,iCode,smen_id)
       		        if err != nil {
                	        panic(err.Error())
                	}
		//Reduce allocated numbers of stock from prod_master too
		insStkupdate, err := db.Prepare("UPDATE prod_master SET qty_in_stock=qty_in_stock - ? WHERE prod_id = ?")
       		if err != nil { panic(err.Error()) }
		insStkupdate.Exec(iQty,iCode)	
       		if err != nil { panic(err.Error()) }
      	  rowCount++;	
          }
	}else{
	
	}
	//Fill Select tag with existtng data
		
	selDBsmen, err := db.Query("SELECT * FROM smen_master WHERE smen_id != 1 ORDER BY smen_id")	
	if err !=nil {
		panic(err.Error())
	}
	selDBprod, err := db.Query("SELECT prod_id,prod_desc,qty_in_stock FROM prod_master ORDER BY prod_id")	
	if err !=nil {
		panic(err.Error())
	}

	//Adding Salesmen details to structure
	smen := Smen{}
	smen_collection := []Smen{}
	for selDBsmen.Next(){
		var id int
		var name string
		err = selDBsmen.Scan(&id, &name)
		if err != nil {
			panic(err.Error())
		}
		smen.Id =id
		smen.Desc = name
		smen_collection = append(smen_collection,smen)

	}
	
	//Adding priduct_id details to struture
        product := Product{}
        product_collection := []Product{}
        for selDBprod.Next(){
                var pro_id int
		var qty int
                var prod_name string
                err = selDBprod.Scan(&pro_id, &prod_name,&qty)
                if err != nil {
                        panic(err.Error())
                }
                product.Id = pro_id
                product.Desc = prod_name
                product.Qty = qty
                product_collection = append(product_collection,product)

        }

//Combining both structures
	var data = struct {
	Sm_Data []Smen
	Prd_Data []Product
}{
	Sm_Data: smen_collection,
	Prd_Data: product_collection,

}

	
    	tmpl.ExecuteTemplate(w, "Smentracker",data)
    	defer db.Close()
}

//Show Reports
func ShowReports(w http.ResponseWriter, r *http.Request){
	//keeping the common part of sql
	strSQL := "SELECT smen_trans.trans_dt, prod_master.prod_desc, smen_trans.qty,smen_master.smen_name FROM smen_master INNER JOIN smen_trans ON smen_master.smen_id = smen_trans.smen_id INNER JOIN prod_master on smen_trans.prod_id = prod_master.prod_id WHERE "
	db := dbConn()
	smen_trans := Smen_trans{} 
	trans_collection := []Smen_trans{}

	//Need to fill sales men details while loaing 
        selDBsmen, err := db.Query("SELECT * FROM smen_master WHERE smen_id != 1 ORDER BY smen_id")
        if err !=nil {
                panic(err.Error())
        }

        //Adding Salesmen details to structure
        smen := Smen{}
        smen_collection := []Smen{}
        for selDBsmen.Next(){
                var id int
                var name string
                err = selDBsmen.Scan(&id, &name)
                if err != nil {
                        panic(err.Error())
                }
                smen.Id =id
                smen.Desc = name
                smen_collection = append(smen_collection,smen)

        }

	if r.Method == "POST"{
		opted_smen_string := r.FormValue("selectSmen") // *** Improve this area first reading it as string to incude in SQL Query then to into to use in compare 
		var opted_smen int
		opted_smen, err = strconv.Atoi(r.FormValue("selectSmen"))
		opted_frmDt := r.FormValue("frm_dt")
		opted_toDt := r.FormValue("to_dt")

		if opted_frmDt == opted_toDt && opted_smen == 0{//Details of a particular daay for all salesmen
			strSQL = fmt.Sprintf("%s%s",strSQL,"smen_trans.trans_dt = '" + opted_frmDt + "' ORDER BY  smen_trans.tans_id")
		}else if opted_frmDt == opted_toDt && opted_smen != 0{// Details of a particular day for any one salesmen
			strSQL = fmt.Sprintf("%s%s",
			strSQL,"smen_trans.smen_id = " + opted_smen_string + " AND smen_trans.trans_dt = '" + opted_frmDt + "' ORDER BY  smen_trans.tans_id")
			
		}else if opted_frmDt != opted_toDt && opted_smen == 0{// Date Range for all salesmen
			strSQL = fmt.Sprintf("%s%s",
			strSQL,"smen_trans.trans_dt BETWEEN '" + opted_frmDt + "' AND '" + opted_toDt + "' ORDER BY  smen_trans.tans_id")

		}else if opted_frmDt != opted_toDt && opted_smen != 0{// Date Range for particular salesmen
			strSQL = fmt.Sprintf("%s%s",
			strSQL,"smen_trans.smen_id = " + opted_smen_string + " AND smen_trans.trans_dt BETWEEN '" + opted_frmDt + "' AND '" + opted_toDt + "' ORDER BY  smen_trans.tans_id")
		}
	}else{//Will fill current date trasactions while loading page  
		strSQL = fmt.Sprintf("%s%s",strSQL,"smen_trans.trans_dt = CURDATE() ORDER BY  smen_trans.tans_id")
	} 
	//fmt.Println(strSQL)
	selDBsmen, err = db.Query(strSQL)

        if err !=nil {
                panic(err.Error())
        }
	//adding details to structure
	for selDBsmen.Next(){
		var dt	string
		var prod string
		var qty int
		var smen string

		err = selDBsmen.Scan(&dt,&prod,&qty,&smen)
		if err != nil {
			panic(err.Error())
		}
		smen_trans.Dt = dt
		smen_trans.Prod = prod
		smen_trans.Qty = qty
		smen_trans.Smen = smen
		trans_collection = append(trans_collection,smen_trans)
	}
	//Combine both transaction and smen structures.
	var combinedData = struct {
	SMenTrans []Smen_trans
	SMenNames []Smen
	}{
	SMenTrans: trans_collection,
	SMenNames: smen_collection, 
	}

	tmpl.ExecuteTemplate(w, "ShowReports",combinedData )
  

}
//Stock Entry
func Stockentry(w http.ResponseWriter, r *http.Request){
	db := dbConn()
       	if r.Method == "POST" {
	rowCount := 1
	for{ // picking data and adding to Product_purcahse till the row ends

		iCodeInputName := fmt.Sprintf("%s%d","itemCodeRow__",rowCount) 
		iQTYInputName := fmt.Sprintf("%s%d","itemQtyRow__",rowCount) 
		iCode := r.FormValue(iCodeInputName)
		iQty := r.FormValue(iQTYInputName)
		purchase_Dt := r.FormValue("items_issueDt")
		if iCode == ""{
			break
		}
		insForm, err := db.Prepare("INSERT INTO prod_purchase(prod_id,dt,qty) VALUES (?,?,?)")
        	if err != nil {
           		panic(err.Error())
        	}
		insForm.Exec(iCode, purchase_Dt, iQty)
        	rowCount++;	

		//Need to update product quantity field in prod_master table too
		insForm, err = db.Prepare("UPDATE prod_master SET qty_in_stock = qty_in_stock + ? WHERE prod_id=?")
        	if err != nil {
           		panic(err.Error())
        	}
		insForm.Exec(iQty,iCode)

	
	}
	}else{
	
	}
	//Fill Select tag with existtng data
		
	selDBsmen, err := db.Query("SELECT * FROM smen_master WHERE smen_id != 1 ORDER BY smen_id")	
	if err !=nil {
		panic(err.Error())
	}
	selDBprod, err := db.Query("SELECT prod_id,prod_desc,qty_in_stock FROM prod_master ORDER BY prod_id")	
	if err !=nil {
		panic(err.Error())
	}

	//Adding Salesmen details to structure
	smen := Smen{}
	smen_collection := []Smen{}
	for selDBsmen.Next(){
		var id int
		var name string
		err = selDBsmen.Scan(&id, &name)
		if err != nil {
			panic(err.Error())
		}
		smen.Id =id
		smen.Desc = name
		smen_collection = append(smen_collection,smen)

	}
	
	//Adding priduct_id details to struture
        product := Product{}
        product_collection := []Product{}
        for selDBprod.Next(){
                var pro_id int
		var qty int
                var prod_name string
                err = selDBprod.Scan(&pro_id, &prod_name,&qty)
                if err != nil {
                        panic(err.Error())
                }
                product.Id = pro_id
                product.Desc = prod_name
                product.Qty = qty
                product_collection = append(product_collection,product)

        }

//Combining both structures
	var data = struct {
	Sm_Data []Smen
	Prd_Data []Product
}{
	Sm_Data: smen_collection,
	Prd_Data: product_collection,

}

	
    	tmpl.ExecuteTemplate(w, "Stockentry",data)
    	defer db.Close()
}

//Expense Report
func ExpenseReports(w http.ResponseWriter, r *http.Request){
	db := dbConn()
	//Common Query String Portion for Expense master
	strSQL := "SELECT expense_master.exp_dt, account_heads.head_desc, expense_master.exp_amt,@total:=@total + expense_master.exp_amt AS cumulativeAmt  FROM expense_master JOIN  account_heads ON expense_master.head_id = account_heads.head_id,(SELECT @total:=0) AS t WHERE "
	if r.Method == "POST" {
		opted_catagory_string := r.FormValue("selectCat") // *** Improve this area first reading it as string to incude in 
								  //SQL Query then to into to use in compare
                var opted_cat int
                opted_cat, err := strconv.Atoi(r.FormValue("selectCat"))
		if err !=nil{
			panic(err.Error())
		}
                opted_frmDt := r.FormValue("frm_dt")
                opted_toDt := r.FormValue("to_dt")
		
		if opted_frmDt == opted_toDt && opted_cat== 0{//Details of a particular day for all catagory 
                        strSQL = fmt.Sprintf("%s%s",strSQL,"expense_master.exp_dt = '" + opted_frmDt + "' ORDER BY  expense_master.head_id")
                }else if opted_frmDt == opted_toDt && opted_cat!= 0{// Details of a particular day for any one catogory
                        strSQL = fmt.Sprintf("%s%s",
                        strSQL,"expense_master.head_id = " + opted_catagory_string + " AND expense_master.exp_dt = '" + opted_frmDt + "' ORDER BY expense_master.head_id")
		}else if opted_frmDt != opted_toDt && opted_cat== 0{// All Expenses for a particualr period
                        strSQL = fmt.Sprintf("%s%s",
                        strSQL,"expense_master.exp_dt BETWEEN '" + opted_frmDt + "' AND '" + opted_toDt + "' ORDER BY  expense_master.head_id")
		}else if opted_frmDt != opted_toDt && opted_cat!= 0{// Date Range - particular catagory
                        strSQL = fmt.Sprintf("%s%s",
                        strSQL,"expense_master.head_id = " + opted_catagory_string + " AND expense_master.exp_dt BETWEEN '" + opted_frmDt + "' AND '" + opted_toDt + "' ORDER BY  expense_master.head_id")
		}
	}else{
		//Need to show current date expanse while loading page
		strSQL = fmt.Sprintf("%s%s",strSQL,"expense_master.exp_dt=CURDATE()  ORDER BY expense_master.head_id;")
		
	}
	
		//Fill select tag with existing catagory details 
		selDBcatagory, err := db.Query("SELECT * FROM account_heads ORDER BY head_desc")
		if err !=nil {
			panic(err.Error())
		}
		catagory := Acc_head{}
		cats_collection := []Acc_head{}
		var head_id int
		var head_desc string
		for selDBcatagory.Next(){
			selDBcatagory.Scan(&head_id, &head_desc)
			if err != nil {
				panic(err.Error())
			}
			catagory.Id = head_id
			catagory.Desc = head_desc
		cats_collection = append(cats_collection,catagory)
		}
		
		selDBexpense, err := db.Query(strSQL)
		if err !=nil{
			panic(err.Error())
		}
		expense := Expenses{}
		expense_collection := []Expenses{}
		var dt,cat string
		var amt,camt float32
		for selDBexpense.Next(){
			selDBexpense.Scan(&dt, &cat, &amt, &camt)
			if err != nil {
				panic(err.Error())
			}
			expense.Dt = dt	
			expense.Desc = cat
			expense.Amt = amt
			expense.Camt = camt
		expense_collection = append(expense_collection,expense)	
		}
		
		//Combine Both structures 	
		var ExpAndCatagoryCombined =  struct{
			AcHead []Acc_head	
			ExpDetails []Expenses
		}{
			AcHead : cats_collection,
			ExpDetails : expense_collection,
		}
			
			
	
    	tmpl.ExecuteTemplate(w, "ExpenseReports",ExpAndCatagoryCombined)
	
defer db.Close()
}

//Sales Operation
func Sales(w http.ResponseWriter, r *http.Request){
	db := dbConn()
	
	if r.Method == "POST" {
		Is_CashSale := "on" == r.FormValue("Sale_Mode_CA")//Is_CashSale will get true or false value
		Is_DirectSale := "on" == r.FormValue("DirectSale")//Is_DirectSale will get true or false value
		Customer_ID := r.FormValue("selectCust")
		Inv_Number := r.FormValue("InvoiceNo") 
		SmenID := r.FormValue("selectSmen") 
	//invoice table updation
	if (Is_DirectSale){// Direct Sale only CustID  & Sale Mode is inserting to DB all others have default valus
		InsInv, err := db.Prepare("INSERT INTO invoice(cust_id,is_cash) VALUES(?,?)")
		if err != nil{ panic(err.Error()) }
		InsInv.Exec(Customer_ID,Is_CashSale)
		if err != nil{ panic(err.Error()) }
	}else{
	//Sale via Salemen(updating invoice table with smen_id too 
                InsInv, err := db.Prepare("INSERT INTO invoice(cust_id,is_cash,smen_id) VALUES(?,?,?)")
                if err != nil{ panic(err.Error()) }
                InsInv.Exec(Customer_ID,Is_CashSale,SmenID)
                if err != nil{ panic(err.Error()) }
	}
		
	rowCount := 1
	for{//Add to DB till rows ends
		iCodeInputName := fmt.Sprintf("%s%d","itemCodeRow__",rowCount)
		iQTYInputName := fmt.Sprintf("%s%d","itemQtyRow__",rowCount)
		iPriceInputName := fmt.Sprintf("%s%d","itemPriceRow__",rowCount)
		iCode := r.FormValue(iCodeInputName)	
		iQty := r.FormValue(iQTYInputName)
		iPrice := r.FormValue(iPriceInputName)
		if iCode == ""{
			break
		}
		//Updating inv_items table
		InsInvItem, err := db.Prepare("INSERT INTO inv_item(inv_id,prod_id,prod_price,prod_qty) VALUES(?,?,?,?)")
		if err != nil{ panic(err.Error()) }
		InsInvItem.Exec(Inv_Number,iCode,iPrice,iQty)
		if err != nil{ panic(err.Error()) }
		
		if(Is_DirectSale){
			//Update Stock in prod_master table
			InsUpdatestock,err := db.Prepare("UPDATE prod_master SET qty_in_stock=qty_in_stock - ? WHERE prod_id=?")
			if err != nil{ panic(err.Error()) }
			InsUpdatestock.Exec(iQty,iCode)
			if err != nil{ panic(err.Error()) }
		}else{// Update required in smen_stock table  
                        InsUpdatestock,err := db.Prepare("UPDATE smen_stock SET assigned_qty=assigned_qty - ? WHERE prod_id=? AND smen_id=?")
                        if err != nil{ panic(err.Error()) }
                        InsUpdatestock.Exec(iQty,iCode,SmenID)
                        if err != nil{ panic(err.Error()) }
		}
		
		
	rowCount++
	}






		
	}else{
		
	}
	//Initally collecting data assigned to smen with fields ( | smen_id | smen_name | prod_id | prod_desc | prod_price | assigned_qty |)
	selDBitems, err := db.Query("SELECT smen_stock.smen_id,smen_master.smen_name,prod_master.prod_id,prod_master.prod_desc,prod_master.prod_price,smen_stock.assigned_qty,prod_master.qty_in_stock FROM  smen_stock JOIN smen_master ON smen_stock.smen_id = smen_master.smen_id INNER JOIN prod_master ON smen_stock.prod_id = prod_master.prod_id;")
	if err != nil {
		panic(err.Error())
	}
	itmFullData := ItmDetais{}
	itemFullCollection := []ItmDetais{}
	var sm_id int
	var sm_nm string
	var prd_id int
	var prd_desc string
	var prd_proce float32
	var prd_qty int
	var prd_whouse_stk int

	for selDBitems.Next(){
		selDBitems.Scan(&sm_id,&sm_nm,&prd_id,&prd_desc,&prd_proce,&prd_qty,&prd_whouse_stk)
		itmFullData.SmenID = sm_id
		itmFullData.SmenName = sm_nm
		itmFullData.ProdID = prd_id
		itmFullData.ProdDesc = prd_desc
		itmFullData.PrdPrce = prd_proce
		itmFullData.PrdQty = prd_qty
		itmFullData.PrdWQty = prd_whouse_stk 
		itemFullCollection = append(itemFullCollection,itmFullData)
	}
	// Generating customer data to pass 
	selDBcustomers, err := db.Query("SELECT cust_id,cust_name FROM customers")
	if err != nil{
		panic(err.Error())
	}
	customer := Acc_head{}
	customer_collection := []Acc_head{} // As of now using this struct , dont confuse with the name Acc_head
	var cust_id int
	var cust_nm string

	for selDBcustomers.Next(){
		selDBcustomers.Scan(&cust_id,&cust_nm)	
		customer.Id = cust_id
		customer.Desc = cust_nm
		customer_collection = append(customer_collection,customer)
	}

	//Generatiing Items List
	selDBprod, err := db.Query("SELECT prod_id,prod_desc,qty_in_stock,prod_price FROM prod_master ORDER BY prod_id")	
	if err !=nil {
		panic(err.Error())
	}
        product := Product{}
        product_collection := []Product{}
        for selDBprod.Next(){
                var pro_id int
                var qty int
                var prod_name string
		var prod_price float32
                err = selDBprod.Scan(&pro_id, &prod_name,&qty,&prod_price)
                if err != nil {
                        panic(err.Error())
                }
                product.Id = pro_id
                product.Desc = prod_name
                product.Qty = qty
                product.Price = prod_price
                product_collection = append(product_collection,product)
        }
	
	//Generating Salesmen Details
        selDBsmen, err := db.Query("SELECT * FROM smen_master WHERE smen_id != 1 ORDER BY smen_id")
        if err !=nil {
                panic(err.Error())
        }
        smen := Smen{}
        smen_collection := []Smen{}
        for selDBsmen.Next(){
                var id int
                var name string
                err = selDBsmen.Scan(&id, &name)
                if err != nil {
                        panic(err.Error())
                }
                smen.Id =id
                smen.Desc = name
                smen_collection = append(smen_collection,smen)

        }

	//Select the last invoice number
	selDBlastINV := db.QueryRow("SELECT inv_id FROM invoice ORDER BY inv_id DESC LIMIT 1")
	var lastINVno int
	switch err = selDBlastINV.Scan(&lastINVno)	
	err{
	case sql.ErrNoRows:
		lastINVno = 1
	case nil:
		lastINVno++
	default:
		panic(err.Error())
	}
	

	//Combining all data

	var combData = struct {
		StkWithSmen []ItmDetais
		Customers []Acc_head
		FullProduct []Product
		SmenData []Smen
		NextInvNo int
	}{
		StkWithSmen: itemFullCollection,
		Customers: customer_collection,
		FullProduct: product_collection,
		SmenData: smen_collection,
		NextInvNo: lastINVno,
	}

	tmpl.ExecuteTemplate(w, "Sales", combData)
	//tmpl.ExecuteTemplate(w, "Sales", itemFullCollection)
}

func SaleReports(w http.ResponseWriter, r *http.Request){
	db := dbConn()	
	//storing the common part to strSQL
	strSQL := "SELECT inv_item.inv_id,smen_master.smen_name,invoice.inv_dt,customers.cust_name,SUM(inv_item.prod_price*inv_item.prod_qty)as inv_amt,CASE WHEN is_cash THEN 'cash' ELSE 'credit' END as sale_mode FROM inv_item INNER JOIN invoice ON invoice.inv_id=inv_item.inv_id INNER JOIN customers on customers.cust_id=invoice.cust_id INNER JOIN smen_master ON invoice.smen_id=smen_master.smen_id WHERE "

	if r.Method == "POST"{
		opted_smen_string := r.FormValue("selectSmen")
		opted_cust_string := r.FormValue("selectCust")
		var opted_smen int
		var opted_cust int
		opted_smen, err := strconv.Atoi(r.FormValue("selectSmen"))
		if err != nil { panic(err.Error()) }
		opted_cust, err = strconv.Atoi(r.FormValue("selectCust"))
		if err != nil { panic(err.Error()) }
	        opted_frmDt := r.FormValue("frm_dt")
                opted_toDt := r.FormValue("to_dt")
		Is_CashSale := "on" == r.FormValue("Sale_Mode_CA")//Is_CashSale will get true or false value
		Is_CreditSale := "on" == r.FormValue("Sale_Mode_CR")//Is_CreditSale will get true or false value
		
		if(Is_CashSale){ strSQL = fmt.Sprintf("%s%s",strSQL,"invoice.is_cash = true AND ") }
		if(Is_CreditSale){ strSQL = fmt.Sprintf("%s%s",strSQL,"invoice.is_cash = false AND ") }

		if opted_smen != 0{ strSQL = fmt.Sprintf("%s%s",strSQL,"smen_master.smen_id = " + opted_smen_string + " AND ") }
		if opted_cust != 0{ strSQL = fmt.Sprintf("%s%s",strSQL,"customers.cust_id = " + opted_cust_string + " AND ") }
		if opted_frmDt == opted_toDt {
			strSQL = fmt.Sprintf("%s%s",strSQL,"invoice.inv_dt = '" + opted_frmDt + "'") 
		}else{
			strSQL = fmt.Sprintf("%s%s",strSQL,"invoice.inv_dt BETWEEN '" + opted_frmDt + "' AND '" + opted_toDt + "'")
		}
		
		
	}else{

		//Showing Current date invoices while page loading..
		strSQL = fmt.Sprintf("%s%s",strSQL,"invoice.inv_dt=CURDATE()")
	}
	//Appedning last part to Query string
	strSQL = fmt.Sprintf("%s%s",strSQL," GROUP BY inv_item.inv_id ORDER BY inv_item.inv_id")
	SelINV, err := db.Query(strSQL)
	if err != nil { panic(err.Error()) }
	invData := InvData{}
	invDataCollection := []InvData{}
	var invno int
	var smen  string
	var invdt string
	var custnm string
	var invamt float32
	var salemod string
	
	for SelINV.Next(){
		err := SelINV.Scan(&invno,&smen,&invdt,&custnm,&invamt,&salemod)
		if err != nil { panic(err.Error()) }
		invData.InvNo = invno
		invData.Smen = smen
		invData.InvDt = invdt
		invData.CustNm = custnm
		invData.InvAmt = invamt
		invData.SaleMod = salemod
		invDataCollection = append(invDataCollection,invData)
		
	}
	//Collecting total SUM of all Invoices
	strSQLdetail := fmt.Sprintf("%s%s","With Detail as (",strSQL + ")SELECT COALESCE(SUM(inv_amt),0) from Detail" )
	var invTotal float32
	selDBinvTotal, err := db.Query(strSQLdetail)
        if err !=nil { panic(err.Error())}
	for selDBinvTotal.Next(){ 
		err = selDBinvTotal.Scan(&invTotal)
        	if err !=nil { panic(err.Error())}
	}
	
	//Customer Data also need to pass for filtering
        selDBcustomer, err := db.Query("SELECT cust_id,cust_name FROM customers;")
        if err !=nil {
                panic(err.Error())
        }
        //Adding Salesmen details to structure
        customer := Smen{} // used same structure for salemen
        customer_collection := []Smen{}
        for selDBcustomer.Next(){
                var id int
                var name string
                err = selDBcustomer.Scan(&id, &name)
                if err != nil {
                        panic(err.Error())
                }
                customer.Id =id
                customer.Desc = name
                customer_collection = append(customer_collection,customer)
	}
        //Adding Salesmen details to structure
        selDBsmen, err := db.Query("SELECT * FROM smen_master ORDER BY smen_id")
        if err !=nil {
                panic(err.Error())
        }

        smendata := Smen{}
        smendata_collection := []Smen{}
        for selDBsmen.Next(){
                var id int
                var name string
                err = selDBsmen.Scan(&id, &name)
                if err != nil {
                        panic(err.Error())
                }
                smendata.Id =id
                smendata.Desc = name
                smendata_collection = append(smendata_collection,smendata)

        }

	//Combin all data
	var comb_data = struct {
		Cust_Data []Smen
		Inv_Data []InvData
		Smen_Data []Smen
		Inv_Total float32
	}{
		Cust_Data: customer_collection,
		Inv_Data: invDataCollection,
		Smen_Data: smendata_collection,
		Inv_Total: invTotal,
	}
	


	tmpl.ExecuteTemplate(w, "SaleReports",comb_data)
	
}

//Showing selected invoice from Invoice report page
func ShowInvoice(w http.ResponseWriter, r *http.Request){

	strSQL := "SELECT inv_item.prod_id,prod_master.prod_desc,inv_item.prod_price,inv_item.prod_qty, (inv_item.prod_price*inv_item.prod_qty) AS subtotal FROM invoice INNER JOIN inv_item ON invoice.inv_id=inv_item.inv_id INNER JOIN smen_master ON smen_master.smen_id=invoice.smen_id INNER JOIN prod_master ON prod_master.prod_id=inv_item.prod_id INNER JOIN customers ON customers.cust_id=invoice.cust_id WHERE invoice.inv_id="

	db := dbConn()
	inv_no := r.URL.Query().Get("invno") //Collecting passed values via href
	strSQL = fmt.Sprintf("%s%s",strSQL,inv_no + " ORDER BY invoice.inv_id") //preparing Quesry with invNo
	selDBinv,err := db.Query(strSQL)
	fmt.Println(strSQL)
	if err !=nil { panic(err.Error())}
	var slnum,proid,pro_qty int
	var prodesc string
	var proprice,amt float32
	itmFullData := ItmDetais{}
	itemFullCollection := []ItmDetais{}
	for selDBinv.Next(){
		err = selDBinv.Scan(&proid,&prodesc,&proprice,&pro_qty,&amt)
		itmFullData.SmenID = slnum // for timebeeing storing serial number to smen id
		itmFullData.ProdID = proid
		itmFullData.ProdDesc = prodesc
		itmFullData.PrdPrce = proprice
		itmFullData.PrdQty = pro_qty
		itmFullData.Extra = amt 
		itemFullCollection = append(itemFullCollection,itmFullData)
	}
	//Using Detail method extracting SUM of item amounts to calcualte the total invoice amount
	strSQLdetail := fmt.Sprintf("%s%s","With Detail as (",strSQL + ")SELECT COALESCE(SUM(subtotal),0) from Detail" )
	var invTotal float32
        selDBinvTotal, err := db.Query(strSQLdetail)
        if err !=nil { panic(err.Error())}
        for selDBinvTotal.Next(){
                err = selDBinvTotal.Scan(&invTotal)
                if err !=nil { panic(err.Error())}
        }
	//Combine invoice items details and total invoice amount
	var combinedData = struct{
		InvItems []ItmDetais
		InvTTL	float32
	}{
		InvItems: itemFullCollection,
		InvTTL: invTotal,
	}

	tmpl.ExecuteTemplate(w,"ShowInvoice",combinedData)	
}
func Settings(w http.ResponseWriter, r *http.Request){
	if r.Method == "POST"{
		db := dbConn() 
		do := r.FormValue("doWhat")
		if "addSmen" == do {
			InsForm, err := db.Prepare("INSERT INTO smen_master(smen_name) VALUES(?)")
        		if err != nil { log.Fatal(err.Error()) }
			InsForm.Exec(r.FormValue("smen"))
		}else if  "addCust" == do{
			InsForm, err := db.Prepare("INSERT INTO customers(cust_name) VALUES(?)")
        		if err != nil { panic(err.Error()) }
		 	InsForm.Exec(r.FormValue("cust"))

		}else if  "addItem" == do{
			InsForm, err := db.Prepare("INSERT INTO prod_master(prod_desc,prod_price) VALUES(?,?)")
        		if err != nil { panic(err.Error()) }
			InsForm.Exec(r.FormValue("item"),r.FormValue("itemprice"))
		}else if "addExpense" == do{
			InsForm, err := db.Prepare("INSERT INTO account_heads(head_desc) VALUES(?)")
        		if err != nil { panic(err.Error()) }
			InsForm.Exec(r.FormValue("expense"))
			
		}
	}
	tmpl.ExecuteTemplate(w, "Settings",nil)
}
func main(){

	//To access css files
	fs := http.FileServer(http.Dir("forms"))
	http.Handle("/",fs)

	log.Println("Server started on localhost:8080")
	http.HandleFunc("/login", Index)
	http.HandleFunc("/chkcredential",Pwd_Verify)
	http.HandleFunc("/sales",Sales)
	http.HandleFunc("/expense",Expense)
	http.HandleFunc("/smentracker",Smentracker)
	http.HandleFunc("/reports",ShowReports)
	http.HandleFunc("/stock",Stockentry)
	http.HandleFunc("/expensereports",ExpenseReports)
	http.HandleFunc("/salereports",SaleReports)
	http.HandleFunc("/showinvoice",ShowInvoice)
	http.HandleFunc("/settings",Settings)

	http.ListenAndServe(":8080",nil)
}
