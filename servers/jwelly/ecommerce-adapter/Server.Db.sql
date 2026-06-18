CREATE TABLE
    `ItemUnit` (
        `itemUnitId` INT NOT NULL,
        `itemUnit` VARCHAR(45) NOT NULL,
        `itemDecimal` INT (1) NULL DEFAULT 0,
        PRIMARY KEY (`itemUnitId`),
        UNIQUE INDEX `itemUnit_UNIQUE` (`itemUnit` ASC) VISIBLE
    );

CREATE TABLE
    `ItemGroup` (
        `itemGroupId` INT NOT NULL,
        `itemGroup` VARCHAR(45) NOT NULL,
        `itemPrintName` VARCHAR(45) NULL,
        `itemGroupUnitId` INT NOT NULL,
        PRIMARY KEY (`itemGroupId`),
        INDEX `IgroupUnitToUnitID_idx` (`itemGroupUnitId` ASC) VISIBLE,
        CONSTRAINT `IgroupUnitToUnitID` FOREIGN KEY (`itemGroupUnitId`) REFERENCES `ItemUnit` (`itemUnitId`) ON DELETE RESTRICT ON UPDATE CASCADE
    );

CREATE TABLE
    `ItemMaster` (
        `itemId` INT NOT NULL,
        `itemName` VARCHAR(45) NULL,
        `iGroupId` INT (3) NULL,
        `itemPrintName` VARCHAR(45) NULL,
        `iUnitId` INT (3) NULL,
        `itemTagPrefix` VARCHAR(5) NULL,
        PRIMARY KEY (`itemId`),
        UNIQUE INDEX `itemName_UNIQUE` (`itemName` ASC) VISIBLE,
        INDEX `ItemMast1_idx` (`iGroupId` ASC) VISIBLE,
        INDEX `ItemMast2_idx` (`iUnitId` ASC) VISIBLE,
        CONSTRAINT `ItemMast1` FOREIGN KEY (`iGroupId`) REFERENCES `ItemGroup` (`itemGroupId`) ON DELETE RESTRICT ON UPDATE CASCADE,
        CONSTRAINT `ItemMast2` FOREIGN KEY (`iUnitId`) REFERENCES `ItemUnit` (`itemUnitId`) ON DELETE RESTRICT ON UPDATE CASCADE
    );

CREATE TABLE
    `Stamp` (
        `stampId` INT NOT NULL,
        `STAMP` VARCHAR(45) NULL,
        `tunch` DECIMAL(8, 2) NULL,
        `stockTunch` DECIMAL(8, 2) NULL,
        `sellTunch` DECIMAL(8, 2) NULL DEFAULT 0,
        PRIMARY KEY (`stampId`)
    );

INSERT INTO
    `Stamp` (
        `stampId`,
        `STAMP`,
        `tunch`,
        `stockTunch`,
        `sellTunch`
    )
VALUES
    ('0', '0', '0', '0', '0');

CREATE TABLE
    `ItemsTag` (
        `itemTagId` INT NOT NULL,
        `itemTag` VARCHAR(45) NOT NULL,
        `itemVTagId` INT NULL,
        `tItemId` INT NULL,
        `tagCreatedDate` DATE NULL,
        PRIMARY KEY (`itemTagId`),
        UNIQUE INDEX `itemTag_UNIQUE` (`itemTag` ASC) VISIBLE,
        UNIQUE INDEX `combo_unique` (`itemVTagId` ASC, `tItemId` ASC) VISIBLE,
        INDEX `itemtag_item_idx` (`tItemId` ASC) VISIBLE,
        CONSTRAINT `itemtag_item` FOREIGN KEY (`tItemId`) REFERENCES `ItemMaster` (`itemId`) ON DELETE RESTRICT ON UPDATE CASCADE
    );

CREATE TABLE
    `ItemTagVariation` (
        `tagVariationId` INT NOT NULL AUTO_INCREMENT,
        `vTagId` INT NULL,
        `vStampId` INT NULL,
        `vGrossWt` DECIMAL(10, 4) NULL,
        `vLessWeight` DECIMAL(10, 4) NULL,
        `vNetWt` DECIMAL(10, 4) NULL,
        `vStatus` BIT (1) NULL,
        `vTunch` DECIMAL(6, 2) NULL,
        `vWstg` DECIMAL(6, 2) NULL,
        `vSellTunch` DECIMAL(6, 2) NULL,
        `vSellWstg` DECIMAL(6, 2) NULL,
        `vKarigarCode` VARCHAR(40) NULL,
        PRIMARY KEY (`tagVariationId`),
        INDEX `TagVariation_1_idx` (`vTagId` ASC) INVISIBLE,
        INDEX `TagVariation_2_idx` (`vStampId` ASC) VISIBLE,
        UNIQUE INDEX `TagVariation_3` (`vTagId` ASC, `vStampId` ASC) INVISIBLE,
        CONSTRAINT `TagVariation_1` FOREIGN KEY (`vTagId`) REFERENCES `ItemsTag` (`itemTagId`) ON DELETE RESTRICT ON UPDATE CASCADE,
        CONSTRAINT `TagVariation_2` FOREIGN KEY (`vStampId`) REFERENCES `Stamp` (`stampId`) ON DELETE RESTRICT ON UPDATE CASCADE
    );

CREATE TABLE
    `ItemTagVariationDetails` (
        `itemTagDetailsId` INT NOT NULL,
        `dItemTagId` INT NOT NULL,
        `dItemId` INT NOT NULL,
        `dSTAMPID` INT NULL DEFAULT 0,
        `dWeight` DECIMAL(10, 3) NULL DEFAULT 0.000,
        `dExWeight` DECIMAL(10, 3) NULL DEFAULT 0.000,
        `dFinalWeight` DECIMAL(10, 3) NULL DEFAULT 0.000,
        `dRemarks` TINYTEXT NULL,
        `dPcs` INT NULL DEFAULT 0,
        `dRate` DECIMAL(12, 2) NULL DEFAULT 0.000,
        `dSaleValue` DECIMAL(12, 2) NULL DEFAULT 0.000,
        `dUnitId` INT NULL,
        PRIMARY KEY (`itemTagDetailsId`),
        INDEX `ItemTagVariationDetails_4_idx` (`dItemTagId` ASC) VISIBLE,
        INDEX `ItemTagVariationDetails_1_idx` (`dSTAMPID` ASC) VISIBLE,
        INDEX `ItemTagVariationDetails_2_idx` (`dItemId` ASC) VISIBLE,
        INDEX `ItemTagVariationDetails_3_idx` (`dUnitId` ASC) VISIBLE,
        CONSTRAINT `ItemTagVariationDetails_1` FOREIGN KEY (`dSTAMPID`) REFERENCES `Stamp` (`stampId`) ON DELETE RESTRICT ON UPDATE CASCADE,
        CONSTRAINT `ItemTagVariationDetails_2` FOREIGN KEY (`dItemId`) REFERENCES `ItemMaster` (`itemId`) ON DELETE RESTRICT ON UPDATE CASCADE,
        CONSTRAINT `ItemTagVariationDetails_4` FOREIGN KEY (`dItemTagId`) REFERENCES `ItemsTag` (`itemTagId`) ON DELETE RESTRICT ON UPDATE CASCADE,
        CONSTRAINT `ItemTagVariationDetails_3` FOREIGN KEY (`dUnitId`) REFERENCES `ItemUnit` (`itemUnitId`) ON DELETE RESTRICT ON UPDATE CASCADE
    );

CREATE TABLE
    `AccountGroup` (
        `groupId` INT NOT NULL,
        `groupName` VARCHAR(80) NOT NULL,
        `underId` INT NOT NULL DEFAULT 0,
        `gType` VARCHAR(45) NULL,
        PRIMARY KEY (`groupId`)
    );

CREATE TABLE
    `AccountMaster` (
        `acno` INT NOT NULL,
        `prefix` VARCHAR(5) DEFAULT "",
        `Name` VARCHAR(60) NOT NULL,
        `pName` VARCHAR(60) DEFAULT "",
        `aGroupId` INT NOT NULL,
        `city` VARCHAR(40) NULL DEFAULT "",
        `location` VARCHAR(30) NULL DEFAULT "",
        `mobile` VARCHAR(20) NULL DEFAULT "",
        `phone` VARCHAR(20) NULL DEFAULT "",
        PRIMARY KEY (`acno`),
        INDEX `AccountMaster_1_idx` (`aGroupId` ASC) VISIBLE,
        CONSTRAINT `AccountMaster_1` FOREIGN KEY (`aGroupId`) REFERENCES `AccountGroup` (`groupId`) ON DELETE RESTRICT ON UPDATE CASCADE
    );